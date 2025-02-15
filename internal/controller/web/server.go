package web

import (
	"context"
	"embed"
	"log/slog"
	"net/http"

	"github.com/jrammler/wheelhouse/internal/service"
)

//go:embed all:static
var staticEmbed embed.FS

type Server struct {
	service *service.Service
	server  *http.Server
}

func NewServer(service *service.Service, addr string) *Server {
	mux := http.NewServeMux()

	staticFs := http.FileServerFS(staticEmbed)
	mux.Handle("/static/", staticFs)

	authenticatedMux := SetupAuthentication(service, mux)
	authenticatedMux.HandleFunc("GET /", handleIndexGet)

	SetupCommandMux(service, authenticatedMux)

	return &Server{
		service: service,
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

func (s *Server) Serve() error {
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) {
	slog.Info("Shutting down server")
	err := s.server.Shutdown(ctx)
	if err != nil {
		slog.Error("Error while shutting down server", "error", err)
	}
	slog.Info("Waiting for all command executions to finish")
	s.service.CommandService.WaitExecutions(ctx)
}

func handleIndexGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/commands", http.StatusFound)
}
