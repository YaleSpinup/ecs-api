package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/ecs-api/orchestration"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

// TaskDefCreateHandler creates the task definition and ensures all of the required
// services exist for running it
func (s *server) TaskDefCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	var req orchestration.TaskDefCreateOrchestrationInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to decode json into input", err))
		return
	}

	log.Debugf("decoded request into taskdef orchestration request: %+v", req)

	output, err := orchestrator.CreateTaskDef(r.Context(), &req)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to marshal response to json", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskDefDeleteHandler handles deleting task definitions and related resources
func (s *server) TaskDefDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	taskdef := vars["taskdef"]

	// Check for the all query param
	recursive := false
	b, err := strconv.ParseBool(r.URL.Query().Get("recursive"))
	if err == nil {
		recursive = b
	}

	force := false
	f, err := strconv.ParseBool(r.URL.Query().Get("force"))
	if err == nil {
		force = f
	}

	log.Debugf("request to delete account %s cluster %s taskdef %s (recursive: %t)", account, cluster, taskdef, recursive)

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	output, err := orchestrator.DeleteTaskDef(r.Context(), &orchestration.TaskDefDeleteInput{
		Cluster:        cluster,
		TaskDefinition: taskdef,
		Recursive:      recursive,
		Force:          force,
	})
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to marshal response to json", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(j)
}

// TaskDefListHandler handles getting a list of task definitions in a cluster
func (s *server) TaskDefListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]

	log.Debugf("listing task definitions in cluster %s", cluster)

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	output, err := orchestrator.ListTaskDefs(r.Context(), cluster)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to marshal response to json", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskDefShowHandler handles getting the details about a task definition in a cluster
func (s *server) TaskDefShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	taskdef := vars["taskdef"]

	log.Debugf("showing taskdef %s/%s/%s", account, cluster, taskdef)

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	output, err := orchestrator.GetTaskDef(r.Context(), cluster, taskdef)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to marshal response to json", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskDefUpdateHandler handles updating a task definition in a cluster
func (s *server) TaskDefUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	taskdef := vars["taskdef"]

	log.Debugf("updating taskdef %s/%s/%s", account, cluster, taskdef)

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	var req orchestration.TaskDefUpdateOrchestrationInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to decode json into input", err))
		return
	}

	log.Debugf("decoded request into taskdef orchestration request:\n %+v", req)

	output, err := orchestrator.UpdateTaskDef(r.Context(), cluster, taskdef, &req)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to marshal response to json", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s *server) TaskDefRunHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	taskdef := vars["taskdef"]

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	var req orchestration.TaskDefRunOrchestrationInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to decode json into input", err))
		return
	}

	log.Debugf("decoded request into taskdef orchestration request: %+v", req)

	output, err := orchestrator.RunTaskDef(r.Context(), cluster, taskdef, req)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to marshal response to json", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s *server) TaskDefTaskListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	taskdef := vars["taskdef"]

	queries := r.URL.Query()

	var startedBy string
	if s, ok := queries["startedBy"]; ok {
		startedBy = s[0]
	}

	status := []string{"RUNNING"}
	if s, ok := queries["status"]; ok {
		status = s
	}

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	output, err := orchestrator.ListTaskDefTasks(r.Context(), cluster, taskdef, startedBy, status)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to marshal response to json", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
