package command

import (
	"bufio"
	"context"
	"errors"
	"github.com/jrammler/wheelhouse/internal/storage"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"sync/atomic"
)

var CommandNotFoundError = errors.New("Command with given ID not found")

type LogEntry struct {
	Stream string
	Data   string
}

type CommandExecution struct {
	ExecId    int
	CommandId int
	ExitCode  *int
	Log       []LogEntry
	logMutex  sync.Mutex
}

type CommandService struct {
	storage       storage.Storage
	execCount     atomic.Uint64
	execWaitGroup *sync.WaitGroup
	history       sync.Map // map of execId to *CommandExecution
}

func (s *CommandService) GetCommands(ctx context.Context) ([]storage.Command, error) {
	config, err := s.storage.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return config.Commands, nil
}

func (e *CommandExecution) pipeToLog(stream string, pipe io.Reader) {
	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			e.logMutex.Lock()
			e.Log = append(e.Log, LogEntry{
				Stream: stream,
				Data:   scanner.Text(),
			})
			e.logMutex.Unlock()
		}
	}()
}

func (s *CommandService) ExecuteCommand(ctx context.Context, id int) (int, error) {
	config, err := s.storage.GetConfig(ctx)
	if err != nil {
		return 0, err
	}
	if id < 0 || id >= len(config.Commands) {
		return 0, CommandNotFoundError
	}
	execId := int(s.execCount.Add(1))
	s.execWaitGroup.Add(1)
	execution := CommandExecution{
		ExecId:    execId,
		CommandId: id,
	}
	s.history.Store(execId, &execution)
	command := config.Commands[id]
	slog.InfoContext(ctx, "Executing command", "execId", execId, "command_id", id, "command_name", command.Name, "command", command.Command)
	cmd := exec.Command("bash", "-c", command.Command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.execWaitGroup.Done()
		return 0, err
	}
	execution.pipeToLog("stdout", stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.execWaitGroup.Done()
		return 0, err
	}
	execution.pipeToLog("stderr", stderr)
	go func() {
		err = cmd.Run()
		if err != nil {
			slog.ErrorContext(ctx, "Command returned error", "execId", execId, "error", err.Error())
		}
		exitCode := cmd.ProcessState.ExitCode()
		execution.ExitCode = &exitCode
		slog.InfoContext(ctx, "Executing command completed", "execId", execId)
		s.execWaitGroup.Done()
	}()
	return execId, nil
}

func (s *CommandService) GetExecution(ctx context.Context, execId int) *CommandExecution {
	e, ok := s.history.Load(execId)
	if !ok {
		return nil
	}
	return e.(*CommandExecution)
}

func (s *CommandService) WaitExecutions(ctx context.Context) {
	s.execWaitGroup.Wait()
}

func NewCommandService(storage storage.Storage) *CommandService {
	s := CommandService{
		storage:       storage,
		execWaitGroup: &sync.WaitGroup{},
	}
	return &s
}
