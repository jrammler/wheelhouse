package main

import (
	"github.com/jrammler/wheelhouse/internal/controller/web"
	"github.com/jrammler/wheelhouse/internal/service"
	"github.com/jrammler/wheelhouse/internal/service/auth"
	"github.com/jrammler/wheelhouse/internal/service/command"
	"github.com/jrammler/wheelhouse/internal/storage"
)

func main() {
	sto := storage.NewJsonStorage("config.json")

	ser := &service.Service{
		CommandService: command.NewCommandService(sto, nil),
		AuthService:    auth.NewAuthService(sto),
	}

	server := web.NewServer(ser)
	server.Serve()
}
