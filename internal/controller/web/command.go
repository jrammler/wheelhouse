package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/jrammler/wheelhouse/internal/controller/web/templates"
)

func (s *Server) AddCommandHandlers() {
	http.HandleFunc("GET /commands", s.handleCommandsGet)
	http.HandleFunc("POST /execute/{id}", s.handleExecutePost)
	http.HandleFunc("GET /executions", s.handleExecutionsGet)
	http.HandleFunc("GET /executions/{id}", s.handleExecutionDetailsGet)
	http.HandleFunc("GET /executions/{id}/log", s.handleExecutionLogGet)
}

func (s *Server) handleCommandsGet(w http.ResponseWriter, r *http.Request) {
	commands, err := s.service.CommandService.GetCommands(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	templates.Commands(commands).Render(r.Context(), w)
}

func (s *Server) handleExecutePost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
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
}

func (s *Server) handleExecutionsGet(w http.ResponseWriter, r *http.Request) {
	history, err := s.service.CommandService.GetExecutionHistory(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	templates.ExecutionList(history).Render(r.Context(), w)
}

func (s *Server) handleExecutionDetailsGet(w http.ResponseWriter, r *http.Request) {
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
}

func (s *Server) handleExecutionLogGet(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	startStr := r.FormValue("start")
	start, err := strconv.Atoi(startStr)
	if err != nil {
		start = 0
	}
	execution := s.service.CommandService.GetExecution(r.Context(), id)
	if execution == nil {
		http.NotFound(w, r)
		return
	}
	templates.LogList(execution, &start).Render(r.Context(), w)
}
