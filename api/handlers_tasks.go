package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// TaskShowHandler gets the details for a task in a cluster
func (s *server) TaskShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	task := vars["task"]

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	output, err := orchestrator.GetTask(r.Context(), cluster, task)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskStopHandler stops a task in a cluster.  Note if this task is managed by a service, it may be restarted.
func (s *server) TaskStopHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	task := vars["task"]

	queries := r.URL.Query()

	var reason string
	if r, ok := queries["reason"]; ok {
		reason = r[0]
	}

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	if err := orchestrator.StopTask(r.Context(), cluster, task, reason); err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
