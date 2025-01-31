package web

import (
	"fmt"
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
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		commands, err := s.service.CommandService.GetCommands(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		templates.Commands(commands).Render(r.Context(), w)
	})
	http.HandleFunc("POST /execute/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		execId, err := s.service.CommandService.ExecuteCommand(r.Context(), id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/execution/%d", execId), http.StatusSeeOther)
	})
	http.HandleFunc("GET /execution/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		execution := s.service.CommandService.GetExecution(r.Context(), id)
		if execution == nil {
			http.NotFound(w, r)
			return
		}
		templates.ExecutionDetails(execution).Render(r.Context(), w)
	})
	http.HandleFunc("GET /execution/{id}/log", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		start, err := strconv.Atoi(r.FormValue("start"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		execution := s.service.CommandService.GetExecution(r.Context(), id)
		if execution == nil {
			http.NotFound(w, r)
			return
		}
		templates.LogList(execution, start).Render(r.Context(), w)
	})
	http.ListenAndServe(":8080", nil)
}
