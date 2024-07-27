package command

import (
	"bytes"
	"context"
	"errors"
	"github.com/jrammler/wheelhouse/internal/storage"
	"log/slog"
	"os/exec"
)

var CommandNotFoundError = errors.New("Command with given ID not found")

type CommandService struct {
	storage storage.Storage
}

func (s *CommandService) GetCommands(ctx context.Context) ([]storage.Command, error) {
	config, err := s.storage.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return config.Commands, nil
}

func (s *CommandService) RunCommand(ctx context.Context, id int) error {
	config, err := s.storage.GetConfig(ctx)
	if err != nil {
		return err
	}
	if id < 0 || id >= len(config.Commands) {
		return CommandNotFoundError
	}
	command := config.Commands[id]
	slog.Info("Running command", "id", id, "name", command.Name, "command", command.Command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command.Command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	slog.Info("Command output", "stderr", stderr.String(), "stdout", stdout.String())
	if err != nil {
		slog.Error("Command returned error", "error", err.Error())
	}
	return nil
}

func NewCommandService(storage storage.Storage) *CommandService {
	s := CommandService{
		storage: storage,
	}
	// ctx := context.Background()
	// s.AddCommand(ctx, "echo \"Hello, World!\"")
	// s.AddCommand(ctx, "echo \"Hello, World!\" 1>&2")
	// s.AddCommand(ctx, "false")
	return &s
}
