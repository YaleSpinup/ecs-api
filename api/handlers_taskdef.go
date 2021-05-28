package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

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

	body, _ := ioutil.ReadAll(r.Body)

	log.Debugf("new taskdef orchestration request body:\n%s", body)

	var req orchestration.TaskDefCreateOrchestrationInput
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		log.Error("cannot Decode body into create taskdef input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	log.Debugf("decoded request into taskdef orchestration request: %+v", req)

	output, err := orchestrator.CreateTaskDef(r.Context(), &req)
	if err != nil {
		log.Errorf("error in creating taskdef orchestration: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
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
	})
	if err != nil {
		log.Errorf("error in taskdef delete orchestration: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
		log.Errorf("error in taskdef list orchestration: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
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
		log.Errorf("error in taskdef get orchestration: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
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

	body, _ := ioutil.ReadAll(r.Body)

	log.Debugf("update taskdef orchestration request body: %s", body)

	var req orchestration.TaskDefUpdateOrchestrationInput
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		log.Error("cannot Decode body into update taskdef input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	log.Debugf("decoded request into taskdef orchestration request:\n %+v", req)

	output, err := orchestrator.UpdateTaskDef(r.Context(), cluster, taskdef, &req)
	if err != nil {
		log.Errorf("error in creating taskdef orchestration: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
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

	body, _ := ioutil.ReadAll(r.Body)

	log.Debugf("run taskdef orchestration request body:\n%s", body)

	var req orchestration.TaskDefRunOrchestrationInput
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		log.Error("cannot Decode body into create taskdef input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	log.Debugf("decoded request into taskdef orchestration request: %+v", req)

	output, err := orchestrator.RunTaskDef(r.Context(), cluster, taskdef, req)
	if err != nil {
		log.Errorf("error in creating taskdef orchestration: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
