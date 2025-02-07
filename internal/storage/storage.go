package storage

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"github.com/jrammler/wheelhouse/internal/entity"
)

var UserNotFoundError = errors.New("User not found")

type config struct {
	Commands []entity.Command
	Users    []entity.User
}

type Storage interface {
	GetCommands(ctx context.Context) ([]entity.Command, error)
	GetUser(ctx context.Context, username string) (entity.User, error)
}

type JsonStorage struct {
	filepath string
}

func NewJsonStorage(filepath string) Storage {
	return &JsonStorage{
		filepath: filepath,
	}
}

func (s *JsonStorage) readConfig() (*config, error) {
	slog.Info("Trying to read config", "path", s.filepath)
	file, err := os.ReadFile(s.filepath)
	if err != nil {
		slog.Error("Error while reading file", "path", s.filepath, "err", err)
		return nil, err
	}
	config := config{}
	err = json.Unmarshal(file, &config)
	if err != nil {
		slog.Error("Error while parsing configuration file", "path", s.filepath, "err", err)
		return nil, err
	}
	slog.Info("Read configuration", "path", s.filepath, "command_count", len(config.Commands), "user_count", len(config.Users))
	return &config, nil
}

func (s *JsonStorage) GetCommands(ctx context.Context) ([]entity.Command, error) {
	config, err := s.readConfig()
	if err != nil {
		return nil, err
	}
	if config.Commands == nil {
		slog.Warn("No Commands found in configuration file", "path", s.filepath)
		return make([]entity.Command, 0), nil
	}
	return config.Commands, nil
}

func (s *JsonStorage) GetUser(ctx context.Context, username string) (entity.User, error) {
	config, err := s.readConfig()
	if err != nil {
		return entity.User{}, err
	}

	for _, user := range config.Users {
		if user.Username == username {
			return user, nil
		}
	}
	return entity.User{}, UserNotFoundError
}
