package main

import (
	"github.com/jrammler/wheelhouse/internal/controller/web"
	"github.com/jrammler/wheelhouse/internal/service"
	"github.com/jrammler/wheelhouse/internal/storage"
)

func main() {
	sto := storage.NewJsonStorage("config.json")
	ser := service.NewService(sto)

	server := web.NewServer(ser)
	server.Serve()
}
