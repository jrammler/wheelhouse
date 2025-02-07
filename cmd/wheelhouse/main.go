package main

import (
	"fmt"
	"log/slog"
	"os"
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
		if len(os.Args) != 3 {
			usageExit()
		}
		serve(os.Args[2])
	case "hash-password":
		hashPassword()
	default:
		usageExit()
	}
}

func usageExit() {
	fmt.Fprintf(os.Stderr, "Usage: %s [serve <addr> | hash-password]\n", os.Args[0])
	os.Exit(1)
}

func serve(addr string) {
	sto := storage.NewJsonStorage("config.json")

	ser := &service.Service{
		CommandService: command.NewCommandService(sto, nil),
		AuthService:    auth.NewAuthService(sto),
	}

	server := web.NewServer(ser)
	err := server.Serve(addr)
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
