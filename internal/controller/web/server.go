package web

import (
	"github.com/jrammler/wheelhouse/internal/service"
	"net/http"
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
	http.HandleFunc("GET /login", s.handleLoginGet)
	http.HandleFunc("POST /login", s.handleLoginPost)
	http.HandleFunc("GET /logout", s.handleLogoutGet)
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

func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
    // TODO: implement this by using the AuthService
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
    // TODO: implement this by using the AuthService
}

func (s *Server) handleLogoutGet(w http.ResponseWriter, r *http.Request) {
    // TODO: implement this by using the AuthService
}
