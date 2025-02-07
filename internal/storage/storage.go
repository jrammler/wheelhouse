package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"log/slog"

	"github.com/jrammler/wheelhouse/internal/entity"
)

var UserNotFoundError = errors.New("User not found")

type config struct {
	Commands []entity.Command `json:"commands"`
	Users    []entity.User    `json:"users"`
}

type Storage interface {
	GetCommands(ctx context.Context) ([]entity.Command, error)
	GetCommandById(ctx context.Context, id string) (*entity.Command, error)
	GetUser(ctx context.Context, username string) (entity.User, error)
	LoadConfig() error
}

type JsonStorage struct {
	filepath       string
	config         *config
	commandsByHash map[string]*entity.Command
	mu             sync.RWMutex
}

func NewJsonStorage(filepath string) (Storage, error) {
	s := &JsonStorage{
		filepath: filepath,
	}
	err := s.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config on startup", "error", err)
		return nil, err
	}
	return s, nil
}

func (s *JsonStorage) LoadConfig() error {
	file, err := os.ReadFile(s.filepath)
	if err != nil {
		slog.Error("Error while reading file", "path", s.filepath, "err", err)
		return err
	}
	cfg := &config{}
	err = json.Unmarshal(file, cfg)
	if err != nil {
		slog.Error("Error while unmarshalling config", "path", s.filepath, "err", err)
		return err
	}

	hashedCommands := make(map[string]*entity.Command)
	for i, command := range cfg.Commands {
		hash := sha256.Sum256([]byte(command.Command))
		hexHash := hex.EncodeToString(hash[:])
		cfg.Commands[i].Id = hexHash
		hashedCommands[hexHash] = &command
	}

	s.mu.Lock()
	s.config = cfg
	s.commandsByHash = hashedCommands
	s.mu.Unlock()
	return nil
}

func (s *JsonStorage) GetCommands(ctx context.Context) ([]entity.Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.config == nil {
		return nil, errors.New("config not loaded")
	}
	if s.config.Commands == nil {
		slog.Warn("No Commands found in configuration file", "path", s.filepath)
		return nil, nil
	}
	return s.config.Commands, nil
}

func (s *JsonStorage) GetCommandById(ctx context.Context, id string) (*entity.Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.commandsByHash == nil {
		return nil, errors.New("config not loaded")
	}
	return s.commandsByHash[id], nil
}

func (s *JsonStorage) GetUser(ctx context.Context, username string) (entity.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.config == nil {
		return entity.User{}, errors.New("config not loaded")
	}

	for _, user := range s.config.Users {
		if user.Username == username {
			return user, nil
		}
	}
	return entity.User{}, UserNotFoundError
}
