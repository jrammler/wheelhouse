package service

import (
	"context"
	"github.com/jrammler/wheelhouse/internal/service/command"
	"github.com/jrammler/wheelhouse/internal/storage"
)

type CommandService interface {
	GetCommands(ctx context.Context) ([]storage.Command, error)
	ExecuteCommand(ctx context.Context, id int) (int, error)
	GetExecution(ctx context.Context, execId int) *command.CommandExecution
	WaitExecutions(ctx context.Context)
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
