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

type lockedCommandExecution struct {
	execution entity.CommandExecution
	logMutex  sync.Mutex
}

type CommandService struct {
	storage       storage.Storage
	execWaitGroup *sync.WaitGroup
	history       []*lockedCommandExecution
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
		history:       make([]*lockedCommandExecution, 0),
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

func pipeStreamToLog(wg *sync.WaitGroup, e *lockedCommandExecution, stream string, pipe io.Reader) {
	wg.Add(1)
	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			e.logMutex.Lock()
			e.execution.Log = append(e.execution.Log, entity.LogEntry{
				Stream: stream,
				Data:   scanner.Text(),
			})
			e.logMutex.Unlock()
		}
		wg.Done()
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

	execution := lockedCommandExecution{
		execution: entity.CommandExecution{
			CommandId: id,
			ExecTime:  time.Now(),
		},
	}

	s.historyMutex.Lock()
	execution.execution.ExecId = len(s.history)
	s.history = append(s.history, &execution)
	s.historyMutex.Unlock()

	slog.Debug("Executing command", "command_id", id, "command_name", command.Name, "command", command.Command)
	cmd := s.commander.Command("bash", "-c", command.Command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	pipeStreamToLog(s.execWaitGroup, &execution, "stdout", stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}
	pipeStreamToLog(s.execWaitGroup, &execution, "stderr", stderr)
	s.execWaitGroup.Add(1)
	go func() {
		err = cmd.Run()
		if err != nil {
			slog.Debug("Command returned error", "error", err)
		}
		exitCode := cmd.ExitCode()
		execution.execution.ExitCode = &exitCode
		slog.Debug("Executing command completed")
		s.execWaitGroup.Done()
	}()
	return execution.execution.ExecId, nil
}

func (s *CommandService) GetExecutionHistory(ctx context.Context, user entity.User) ([]entity.ExecutionHistoryEntry, error) {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	history := make([]entity.ExecutionHistoryEntry, 0)
	for i, execution := range s.history {
		command, err := s.storage.GetCommandById(ctx, execution.execution.CommandId)
		if err != nil || command == nil {
			slog.Error("command not found", "command_id", execution.execution.CommandId)
			continue
		}

		if command.Role != nil && !userHasRole(user, *command.Role) {
			continue
		}

		history = append(history, entity.ExecutionHistoryEntry{
			ExecId:      i,
			Time:        execution.execution.ExecTime,
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
	execution := &s.history[execId].execution

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
