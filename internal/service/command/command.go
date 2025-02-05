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
	"sync/atomic"
	"time"

	"github.com/jrammler/wheelhouse/internal/entity"
	"github.com/jrammler/wheelhouse/internal/storage"
)

var CommandNotFoundError = errors.New("Command with given ID not found")

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
	execCount     atomic.Uint64
	execWaitGroup *sync.WaitGroup
	history       sync.Map // map of execId to *lockedCommandExecution
	commander     Commander
}

func NewCommandService(storage storage.Storage, commander Commander) *CommandService {
	if commander == nil {
		commander = &execCommander{}
	}
	s := CommandService{
		storage:       storage,
		execWaitGroup: &sync.WaitGroup{},
		commander:     commander,
	}
	return &s
}

func (s *CommandService) GetCommands(ctx context.Context) ([]entity.Command, error) {
	commands, err := s.storage.GetCommands(ctx)
	if err != nil {
		return nil, err
	}
	return commands, nil
}

func pipeStreamToLog(wg *sync.WaitGroup, e *lockedCommandExecution, stream string, pipe io.Reader) {
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

func (s *CommandService) ExecuteCommand(ctx context.Context, id int) (int, error) {
	commands, err := s.storage.GetCommands(ctx)
	if err != nil {
		return 0, err
	}
	if id < 0 || id >= len(commands) {
		return 0, CommandNotFoundError
	}
	execId := int(s.execCount.Add(1)) - 1
	// the counter is incremented by 3, because we have to wait for three things
	// - the command execution itself
	// - reading everything from stdout
	// - reading everything from stderr
	s.execWaitGroup.Add(3)
	execution := lockedCommandExecution{
		execution: entity.CommandExecution{
			ExecId:    execId,
			CommandId: id,
			ExecTime:  time.Now(),
		},
	}
	s.history.Store(execId, &execution)
	command := commands[id]
	slog.InfoContext(ctx, "Executing command", "execId", execId, "command_id", id, "command_name", command.Name, "command", command.Command)
	cmd := s.commander.Command("bash", "-c", command.Command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.execWaitGroup.Done()
		return 0, err
	}
	pipeStreamToLog(s.execWaitGroup, &execution, "stdout", stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.execWaitGroup.Done()
		return 0, err
	}
	pipeStreamToLog(s.execWaitGroup, &execution, "stderr", stderr)
	go func() {
		err = cmd.Run()
		if err != nil {
			slog.InfoContext(ctx, "Command returned error", "execId", execId, "error", err.Error())
		}
		exitCode := cmd.ExitCode()
		execution.execution.ExitCode = &exitCode
		slog.InfoContext(ctx, "Executing command completed", "execId", execId)
		s.execWaitGroup.Done()
	}()
	return execId, nil
}

func (s *CommandService) GetExecutionHistory(ctx context.Context) ([]entity.ExecutionHistoryEntry, error) {
	commands, err := s.storage.GetCommands(ctx)
	if err != nil {
		return nil, err
	}
	var history []entity.ExecutionHistoryEntry
	s.history.Range(func(key any, value any) bool {
		execId := key.(int)
		execution := value.(*lockedCommandExecution).execution
		history = append(history, entity.ExecutionHistoryEntry{
			ExecId:      execId,
			Time:        execution.ExecTime,
			CommandName: commands[execution.CommandId].Name,
		})
		return true
	})
    slices.SortFunc(history, func(a, b entity.ExecutionHistoryEntry) int {
        return int(a.Time.Sub(b.Time).Milliseconds())
    })
	return history, nil
}

func (s *CommandService) GetExecution(ctx context.Context, execId int) *entity.CommandExecution {
	e, ok := s.history.Load(execId)
	if !ok {
		return nil
	}
	return &e.(*lockedCommandExecution).execution
}

func (s *CommandService) WaitExecutions(ctx context.Context) {
	s.execWaitGroup.Wait()
}
