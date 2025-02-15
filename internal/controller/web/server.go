package web

import (
	"embed"
	"net/http"

	"github.com/jrammler/wheelhouse/internal/service"
)

//go:embed all:static
var staticEmbed embed.FS

type Server struct {
	service *service.Service
}

func NewServer(service *service.Service) *Server {
	return &Server{
		service: service,
	}
}

func (s *Server) Serve(addr string) error {
	mux := http.NewServeMux()

	staticFs := http.FileServerFS(staticEmbed)
	mux.Handle("/static/", staticFs)

	authenticatedMux := SetupAuthentication(s.service, mux)
	authenticatedMux.HandleFunc("GET /", s.handleIndexGet)

	SetupCommandMux(s.service, authenticatedMux)

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleIndexGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/commands", http.StatusFound)
}
