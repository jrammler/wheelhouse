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
		http.Redirect(w, r, "/commands", http.StatusMovedPermanently)
	})
	http.HandleFunc("GET /commands", func(w http.ResponseWriter, r *http.Request) {
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
		http.Redirect(w, r, fmt.Sprintf("/executions/%d", execId), http.StatusSeeOther)
	})
	http.HandleFunc("GET /executions", func(w http.ResponseWriter, r *http.Request) {
		history, err := s.service.CommandService.GetExecutionHistory(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		templates.ExecutionList(history).Render(r.Context(), w)
	})
	http.HandleFunc("GET /executions/{id}", func(w http.ResponseWriter, r *http.Request) {
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
	http.HandleFunc("GET /executions/{id}/log", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		start, err := strconv.Atoi(r.FormValue("start"))
		if err != nil {
			start = 0
		}
		execution := s.service.CommandService.GetExecution(r.Context(), id)
		if execution == nil {
			http.NotFound(w, r)
			return
		}
		templates.LogList(execution, &start).Render(r.Context(), w)
	})
	http.ListenAndServe(":8080", nil)
}
