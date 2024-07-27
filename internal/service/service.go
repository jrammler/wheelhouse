package service

import (
	"context"
	"github.com/jrammler/wheelhouse/internal/service/command"
	"github.com/jrammler/wheelhouse/internal/storage"
)

type CommandService interface {
	GetCommands(ctx context.Context) ([]storage.Command, error)
	RunCommand(ctx context.Context, id int) error
}

type Service struct {
	CommandService CommandService
}

func NewService(storage storage.Storage) *Service {
	commandService := command.NewCommandService(storage)
	return &Service{
		CommandService: commandService,
	}
}
