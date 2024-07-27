package web

import (
	"github.com/jrammler/wheelhouse/internal/controller/web/templates"
	"github.com/jrammler/wheelhouse/internal/service"
	"net/http"
	"strconv"
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
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		commands, err := s.service.CommandService.GetCommands(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		templates.Commands(commands).Render(r.Context(), w)
	})
	http.HandleFunc("POST /run_command/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.service.CommandService.RunCommand(r.Context(), id)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	http.ListenAndServe(":8080", nil)
}
