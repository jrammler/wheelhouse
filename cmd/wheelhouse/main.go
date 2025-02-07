package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jrammler/wheelhouse/internal/controller/web"
	"github.com/jrammler/wheelhouse/internal/service"
	"github.com/jrammler/wheelhouse/internal/service/auth"
	"github.com/jrammler/wheelhouse/internal/service/command"
	"github.com/jrammler/wheelhouse/internal/storage"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) <= 1 {
		usageExit()
	}
	switch os.Args[1] {
	case "serve":
		if len(os.Args) < 4 {
			usageExit()
		}
		serve(os.Args[2], os.Args[3])
	case "hash-password":
		hashPassword()
	default:
		usageExit()
	}
}

func usageExit() {
	fmt.Fprintf(os.Stderr, "Usage: %s [serve <addr> <config-file> | hash-password]\n", os.Args[0])
	os.Exit(1)
}

func serve(addr string, storagePath string) {
	sto, err := storage.NewJsonStorage(storagePath)
	if err != nil {
		slog.Error("Error initializing storage", "error", err)
		os.Exit(1)
	}

	// Set up signal handling for config reload
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP) // Listen for SIGHUP
	go func() {
		for sig := range signalChan {
			slog.Info("Received signal", "signal", sig)
			err := sto.LoadConfig()
			if err != nil {
				slog.Error("Failed to reload config. Continuing with previous config", "error", err)
			} else {
				slog.Info("Config reloaded successfully")
			}
		}
	}()

	ser := &service.Service{
		CommandService: command.NewCommandService(sto, nil),
		AuthService:    auth.NewAuthService(sto),
	}

	server := web.NewServer(ser)
	err = server.Serve(addr)
	if err != nil {
		slog.Error("Error reading password", "error", err)
		os.Exit(1)
	}
}

func hashPassword() {
	fmt.Print("Enter password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		slog.Error("Error reading password", "error", err)
		os.Exit(1)
	}
	password := string(bytePassword)

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("Error hashing password", "error", err)
		os.Exit(1)
	}

	fmt.Printf("\nHashed password: %s\n", string(hashedPassword))
}
