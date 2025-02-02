package command

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jrammler/wheelhouse/internal/entity"
	"github.com/jrammler/wheelhouse/internal/storage"
)

var CommandNotFoundError = errors.New("Command with given ID not found")

type CommandService struct {
	storage       storage.Storage
	execCount     atomic.Uint64
	execWaitGroup *sync.WaitGroup
	history       sync.Map // map of execId to *lockedCommandExecution
}

type lockedCommandExecution struct {
	execution entity.CommandExecution
	logMutex  sync.Mutex
}

func (s *CommandService) GetCommands(ctx context.Context) ([]entity.Command, error) {
	commands, err := s.storage.GetCommands(ctx)
	if err != nil {
		return nil, err
	}
	return commands, nil
}

func pipeStreamToLog(e *lockedCommandExecution, stream string, pipe io.Reader) {
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
	s.execWaitGroup.Add(1)
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
	cmd := exec.Command("bash", "-c", command.Command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.execWaitGroup.Done()
		return 0, err
	}
	pipeStreamToLog(&execution, "stdout", stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.execWaitGroup.Done()
		return 0, err
	}
	pipeStreamToLog(&execution, "stderr", stderr)
	go func() {
		err = cmd.Run()
		if err != nil {
			slog.InfoContext(ctx, "Command returned error", "execId", execId, "error", err.Error())
		}
		exitCode := cmd.ProcessState.ExitCode()
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

func NewCommandService(storage storage.Storage) *CommandService {
	s := CommandService{
		storage:       storage,
		execWaitGroup: &sync.WaitGroup{},
	}
	return &s
}
