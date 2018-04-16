package main

import (
	"encoding/json"
	"net/http"

	"git.yale.edu/spinup/ecs-api/ecs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// TaskDefCreateHandler creates a task definition
func TaskDefCreateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var taskDefRequest ecs.TaskDefReq
	err := json.NewDecoder(r.Body).Decode(&taskDefRequest)
	if err != nil {
		log.Error("cannot decode body into create taskdef request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	log.Debugf("decoded request into task def request: %+v", taskDefRequest)

	resp, err := ecsService.CreateTaskDef(r.Context(), taskDefRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", resp, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskDefListHandler gets a list of task definitions
func TaskDefListHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Allow setting a status (ACTIVE/INACTIVE) as a query parameter.  Set 'ACTIVE' by default
	q := r.URL.Query()
	status := q.Get("status")
	if status == "" {
		status = "ACTIVE"
	}

	list, err := ecsService.ListTaskDefs(r.Context(), status)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(list)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", list, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskDefShowHandler gets a list of task definitions
func TaskDefShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	taskdef := vars["taskdef"]
	ecsService, ok := EcsServices[account]
	if !ok || taskdef == "" {
		log.Errorf("account or taskdef not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := ecsService.GetTaskDef(r.Context(), taskdef)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", resp, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskDefDeleteHandler deregisters a task definition
func TaskDefDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	taskdef := vars["taskdef"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := ecsService.DeleteTaskDef(r.Context(), taskdef)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", resp, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
