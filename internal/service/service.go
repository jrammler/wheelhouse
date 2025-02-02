package service

import (
	"context"
	"time"

	"github.com/jrammler/wheelhouse/internal/entity"
)

type CommandService interface {
	GetCommands(ctx context.Context) ([]entity.Command, error)
	ExecuteCommand(ctx context.Context, id int) (int, error)
	GetExecutionHistory(ctx context.Context) ([]entity.ExecutionHistoryEntry, error)
	GetExecution(ctx context.Context, execId int) *entity.CommandExecution
	WaitExecutions(ctx context.Context)
}

type AuthService interface {
	LoginUser(ctx context.Context, username, password string) (sessionToken string, expiration *time.Time, err error)
	LogoutUser(ctx context.Context, sessionToken string)
	GetSessionUser(ctx context.Context, sessionToken string) (user entity.User, err error)
}

type Service struct {
	CommandService CommandService
	AuthService    AuthService
}
