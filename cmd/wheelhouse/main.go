package main

import (
	"context"
	"errors"
	"github.com/jrammler/wheelhouse/internal/service"
	"github.com/jrammler/wheelhouse/internal/service/command"
	"github.com/jrammler/wheelhouse/internal/storage"
	"log/slog"
)

func main() {
	sto := storage.NewJsonStorage("config.json")
	ser := service.NewService(sto)

	ctx := context.Background()
	commands, err := ser.CommandService.GetCommands(ctx)
	if err != nil {
		slog.Error("Could not load command list", "reason", err)
		return
	}
	slog.Info("Commands loaded", "commands", commands)
	for i := 0; err == nil; i++ {
		err = ser.CommandService.RunCommand(ctx, i)
	}
	if !errors.Is(err, command.CommandNotFoundError) {
		slog.Info("Iteration of commands stopped", "reason", err)
	}
	ser.CommandService.(*command.CommandService).WaitGroup.Wait()
}
