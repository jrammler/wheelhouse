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

type Commander interface {
	Command(name string, arg ...string) Command
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

type execCommander struct{}

func (rc *execCommander) Command(name string, arg ...string) Command {
	return &execCommand{
		cmd: exec.Command(name, arg...),
	}
}

type CommandService struct {
	storage       storage.Storage
	execWaitGroup *sync.WaitGroup
	history       []*entity.CommandExecution
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

func pipeStreamToLog(e *entity.CommandExecution, stream string, pipe io.Reader, logChan chan<- entity.LogEntry, doneChan chan<- int) {
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
	execution.ExecId = len(s.history)
	s.history = append(s.history, &execution)
	s.historyMutex.Unlock()

	slog.Debug("Executing command", "command_id", id, "command_name", command.Name, "command", command.Command)
	cmd := s.commander.Command("bash", "-c", command.Command)

	logChan := make(chan entity.LogEntry)
	doneChan := make(chan int)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	pipeStreamToLog(&execution, "stdout", stdout, logChan, doneChan)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}
	pipeStreamToLog(&execution, "stderr", stderr, logChan, doneChan)

	allDone := make(chan any)
	go func() {
		doneCnt := 0
        // read from log channel until both stdout and stderr are closed
		for doneCnt < 2 {
			select {
			case log := <-logChan:
				execution.Log = append(execution.Log, log)
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
			slog.Debug("Command returned error", "error", err)
		}
        // only set exit code once log is fully written
		<-allDone
		exitCode := cmd.ExitCode()
		execution.ExitCode = &exitCode
		slog.Debug("Executing command completed")
		s.execWaitGroup.Done()
	}()
	return execution.ExecId, nil
}

func (s *CommandService) GetExecutionHistory(ctx context.Context, user entity.User) ([]entity.ExecutionHistoryEntry, error) {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	history := make([]entity.ExecutionHistoryEntry, 0)
	for i, execution := range s.history {
		command, err := s.storage.GetCommandById(ctx, execution.CommandId)
		if err != nil || command == nil {
			slog.Error("command not found", "command_id", execution.CommandId)
			continue
		}

		if command.Role != nil && !userHasRole(user, *command.Role) {
			continue
		}

		history = append(history, entity.ExecutionHistoryEntry{
			ExecId:      i,
			Time:        execution.ExecTime,
			CommandName: command.Name,
		})
	}

	slices.SortFunc(history, func(a, b entity.ExecutionHistoryEntry) int {
		return int(a.Time.Sub(b.Time).Milliseconds())
	})
	return history, nil
}

func (s *CommandService) GetExecution(ctx context.Context, user entity.User, execId int) (*entity.CommandExecution, error) {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	if execId < 0 || execId >= len(s.history) {
		return nil, CommandNotFoundError
	}
	execution := s.history[execId]

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
	s.execWaitGroup.Wait()
}

func userHasRole(user entity.User, role string) bool {
	return slices.Contains(user.Roles, role)
}
