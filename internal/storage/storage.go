package storage

import (
	"context"
	"encoding/json"
	"os"
)

type Command struct {
	Name    string
	Command string
}

type Config struct {
	Commands []Command
}

type Storage interface {
	GetConfig(ctx context.Context) (*Config, error)
}

type JsonStorage struct {
	filepath string
}

func NewJsonStorage(filepath string) *JsonStorage {
	return &JsonStorage{
		filepath: filepath,
	}
}

func (s *JsonStorage) GetConfig(ctx context.Context) (*Config, error) {
	file, err := os.ReadFile(s.filepath)
	if err != nil {
		return nil, err
	}
	config := Config{}
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
