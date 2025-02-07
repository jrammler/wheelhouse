package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/jrammler/wheelhouse/internal/controller/web/templates"
	"github.com/jrammler/wheelhouse/internal/service"
)

func SetupCommandMux(service *service.Service, mux *http.ServeMux) {
	mux.HandleFunc("GET /commands", handleCommandsGet(service))
	mux.HandleFunc("POST /execute/{id}", handleExecutePost(service))
	mux.HandleFunc("GET /executions", handleExecutionsGet(service))
	mux.HandleFunc("GET /executions/{id}", handleExecutionDetailsGet(service))
	mux.HandleFunc("GET /executions/{id}/log", handleExecutionLogGet(service))
}

func handleCommandsGet(service *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		commands, err := service.CommandService.GetCommands(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		templates.Commands(commands).Render(r.Context(), w)
	}
}

func handleExecutePost(service *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		execId, err := service.CommandService.ExecuteCommand(r.Context(), id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/executions/%d", execId), http.StatusFound)
	}
}

func handleExecutionsGet(service *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		history, err := service.CommandService.GetExecutionHistory(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		templates.ExecutionList(history).Render(r.Context(), w)
	}
}

func handleExecutionDetailsGet(service *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		execution := service.CommandService.GetExecution(r.Context(), id)
		if execution == nil {
			http.NotFound(w, r)
			return
		}
		templates.ExecutionDetails(execution).Render(r.Context(), w)
	}
}

func handleExecutionLogGet(service *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		execution := service.CommandService.GetExecution(r.Context(), id)
		if execution == nil {
			http.NotFound(w, r)
			return
		}
		templates.LogList(execution, &start).Render(r.Context(), w)
	}
}
