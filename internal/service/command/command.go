package command

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"slices"
	"sync"
	"time"

	"github.com/jrammler/wheelhouse/internal/entity"
	"github.com/jrammler/wheelhouse/internal/storage"
)

var CommandNotFoundError = errors.New("Command with given ID not found")
var UnauthorizedError = errors.New("User is not authorized to execute this command")

type Command interface {
	Run() error
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	ExitCode() int
}

type execCommand struct {
	cmd *exec.Cmd
}

func (e *execCommand) Run() error {
	return e.cmd.Run()
}

func (e *execCommand) StdoutPipe() (io.ReadCloser, error) {
	return e.cmd.StdoutPipe()
}

func (e *execCommand) StderrPipe() (io.ReadCloser, error) {
	return e.cmd.StderrPipe()
}

func (e *execCommand) ExitCode() int {
	if e.cmd.ProcessState != nil {
		return e.cmd.ProcessState.ExitCode()
	}
	return -1
}

type CommandService struct {
	storage       storage.Storage
	execWaitGroup *sync.WaitGroup
	history       []*entity.CommandExecution
	historyOffset int
	historyMutex  sync.RWMutex
	commander     Commander
}

func NewCommandService(storage storage.Storage, commander Commander) *CommandService {
	if commander == nil {
		commander = &execCommander{}
	}
	s := CommandService{
		storage:       storage,
		execWaitGroup: &sync.WaitGroup{},
		history:       make([]*entity.CommandExecution, 0),
		commander:     commander,
	}
	return &s
}

func (s *CommandService) GetCommands(ctx context.Context, user entity.User) ([]entity.Command, error) {
	commands, err := s.storage.GetCommands(ctx)
	if err != nil {
		return nil, err
	}

	filteredCommands := make([]entity.Command, 0)
	for _, command := range commands {
		if command.Role == nil || userHasRole(user, *command.Role) {
			filteredCommands = append(filteredCommands, command)
		}
	}

	return filteredCommands, nil
}

func pipeStreamToLog(stream string, pipe io.Reader, logChan chan<- entity.LogEntry, doneChan chan<- int) {
	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			logChan <- entity.LogEntry{
				Stream: stream,
				Data:   scanner.Text(),
			}
		}
		doneChan <- 0
	}()
}

const maxHistLen int = 100
const maxLogLen int = 1000

func (s *CommandService) ExecuteCommand(ctx context.Context, user entity.User, id string) (int, error) {
	command, err := s.storage.GetCommandById(ctx, id)
	if err != nil {
		return 0, err
	}
	if command == nil {
		return 0, CommandNotFoundError
	}

	if command.Role != nil && !userHasRole(user, *command.Role) {
		return 0, UnauthorizedError
	}

	execution := entity.CommandExecution{
		CommandId: id,
		ExecTime:  time.Now(),
	}

	s.historyMutex.Lock()
	histLen := len(s.history)
	execution.ExecId = histLen + s.historyOffset
	if histLen >= maxHistLen {
		removeCnt := histLen - maxHistLen + 1
		for i := 0; i < removeCnt; i++ {
			s.history[i] = nil
		}
		s.history = s.history[removeCnt:]
		s.historyOffset += removeCnt
	}
	s.history = append(s.history, &execution)
	s.historyMutex.Unlock()

	slog.Info("Executing command", "command_id", id, "command_name", command.Name, "command", command.Command)
	cmd := s.commander.Command(command.Command)

	logChan := make(chan entity.LogEntry)
	doneChan := make(chan int)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	pipeStreamToLog("stdout", stdout, logChan, doneChan)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}
	pipeStreamToLog("stderr", stderr, logChan, doneChan)

	allDone := make(chan any)
	go func() {
		doneCnt := 0
		// read from log channel until both stdout and stderr are closed
	loop:
		for doneCnt < 2 {
			select {
			case log := <-logChan:
				execution.Log = append(execution.Log, log)
				if len(execution.Log) > maxLogLen {
					execution.Log = append(execution.Log, entity.LogEntry{
						Stream: "system",
						Data:   "log truncated ...",
					})
					break loop
				}
			case <-doneChan:
				doneCnt += 1
			}
		}
		close(allDone)
	}()

	s.execWaitGroup.Add(1)
	go func() {
		err = cmd.Run()
		if err != nil {
			slog.Info("Command returned error", "error", err)
		}
		// only set exit code once log is fully written
		<-allDone
		exitCode := cmd.ExitCode()
		execution.ExitCode = &exitCode
		slog.Info("Executing command completed")
		s.execWaitGroup.Done()
	}()
	return execution.ExecId, nil
}

func (s *CommandService) GetExecutionHistory(ctx context.Context, user entity.User) ([]entity.ExecutionHistoryEntry, error) {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	history := make([]entity.ExecutionHistoryEntry, 0)
	for _, execution := range s.history {
		command, err := s.storage.GetCommandById(ctx, execution.CommandId)
		if err != nil || command == nil {
			slog.Error("command not found", "command_id", execution.CommandId)
			continue
		}

		if command.Role != nil && !userHasRole(user, *command.Role) {
			continue
		}

		history = append(history, entity.ExecutionHistoryEntry{
			ExecId:      execution.ExecId,
			Time:        execution.ExecTime,
			CommandName: command.Name,
			ExitCode:    execution.ExitCode,
		})
	}

	return history, nil
}

func (s *CommandService) GetExecution(ctx context.Context, user entity.User, execId int) (*entity.CommandExecution, error) {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	idx := execId - s.historyOffset
	if idx < 0 || idx >= len(s.history) {
		return nil, CommandNotFoundError
	}
	execution := s.history[idx]

	command, err := s.storage.GetCommandById(ctx, execution.CommandId)
	if err != nil || command == nil {
		slog.Error("command not found", "command_id", execution.CommandId)
		return nil, CommandNotFoundError
	}

	if command.Role != nil && !userHasRole(user, *command.Role) {
		return nil, UnauthorizedError
	}

	return execution, nil
}

func (s *CommandService) WaitExecutions(ctx context.Context) {
	done := make(chan any)
	go func() {
		s.execWaitGroup.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func userHasRole(user entity.User, role string) bool {
	return slices.Contains(user.Roles, role)
}
