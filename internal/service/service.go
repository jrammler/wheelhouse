package service

import (
	"context"
	"time"

	"github.com/jrammler/wheelhouse/internal/entity"
)

type CommandService interface {
	GetCommands(ctx context.Context, user entity.User) ([]entity.Command, error)
	ExecuteCommand(ctx context.Context, user entity.User, id string) (int, error)
	GetExecutionHistory(ctx context.Context, user entity.User) ([]entity.ExecutionHistoryEntry, error)
	GetExecution(ctx context.Context, user entity.User, execId int) (*entity.CommandExecution, error)
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
