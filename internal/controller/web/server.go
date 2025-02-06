package web

import (
	"net/http"

	"github.com/jrammler/wheelhouse/internal/service"
)

type Server struct {
	service *service.Service
}

func NewServer(service *service.Service) *Server {
	return &Server{
		service: service,
	}
}

func (s *Server) Serve() {
	http.HandleFunc("GET /", s.handleHomeGet)
	s.AddAuthHandlers()
	s.AddCommandHandlers()
	http.ListenAndServe(":8080", nil)
}

func (s *Server) handleHomeGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/commands", http.StatusMovedPermanently)
}
