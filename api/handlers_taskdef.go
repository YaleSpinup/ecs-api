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

	log.Debugf("decoded request into taskdef orchestration request:\n %+v", req)

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
