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

type CommandService struct {
	storage   storage.Storage
	WaitGroup *sync.WaitGroup
	runCount  atomic.Uint64
}

func (s *CommandService) GetCommands(ctx context.Context) ([]storage.Command, error) {
	config, err := s.storage.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return config.Commands, nil
}

func pipeToLog(ctx context.Context, runId uint64, stream string, pipe io.Reader) {
	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			slog.InfoContext(ctx, "Command output", "run_id", runId, "stream", stream, "output", scanner.Text())
		}
	}()
}

func (s *CommandService) RunCommand(ctx context.Context, id int) error {
	config, err := s.storage.GetConfig(ctx)
	if err != nil {
		return err
	}
	if id < 0 || id >= len(config.Commands) {
		return CommandNotFoundError
	}
	runId := s.runCount.Add(1)
	s.WaitGroup.Add(1)
	command := config.Commands[id]
	slog.InfoContext(ctx, "Running command", "run_id", runId, "command_id", id, "command_name", command.Name, "command", command.Command)
	cmd := exec.Command("bash", "-c", command.Command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	pipeToLog(ctx, runId, "stdout", stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	pipeToLog(ctx, runId, "stderr", stderr)
	go func() {
		err = cmd.Run()
		if err != nil {
			slog.ErrorContext(ctx, "Command returned error", "run_id", runId, "error", err.Error())
		}
		slog.InfoContext(ctx, "Running command completed", "run_id", runId)
		s.WaitGroup.Done()
	}()
	return nil
}

func NewCommandService(storage storage.Storage) *CommandService {
	s := CommandService{
		storage:   storage,
		WaitGroup: &sync.WaitGroup{},
	}
	// ctx := context.Background()
	// s.AddCommand(ctx, "echo \"Hello, World!\"")
	// s.AddCommand(ctx, "echo \"Hello, World!\" 1>&2")
	// s.AddCommand(ctx, "false")
	return &s
}
