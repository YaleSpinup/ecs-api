package main

import (
	"encoding/json"
	"net/http"
	"time"

	"git.yale.edu/spinup/ecs-api/ecs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// TaskListHandler gets a list of tasks in a cluster
func TaskListHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account or cluster not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tasks, err := ecsService.ListTasks(r.Context(), cluster)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(tasks)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", tasks, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskDeleteHandler stops a task in a cluster
func TaskDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	task := vars["task"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := ecsService.DeleteTask(r.Context(), cluster, task)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("Cannot marshal response (%v) into JSON: %s", resp, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// TaskShowHandler gets the details for a task in a cluster
func TaskShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	task := vars["task"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if task == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := ecsService.GetTask(r.Context(), cluster, task)
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

// TaskCreateHandler runs a task in a cluster
func TaskCreateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var taskRequest ecs.TaskRequest
	err := json.NewDecoder(r.Body).Decode(&taskRequest)
	if err != nil {
		log.Error("cannot Decode body into create task request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	log.Debugf("decoded request into task request: %+v", taskRequest)

	var task *ecs.Task

	// Wait for task to start
	q := r.URL.Query()
	wait := q.Get("wait")
	if wait != "" {
		wt, err := time.ParseDuration(wait)
		if err != nil {
			log.Warnf("failed parsing wait time from URL, using default 60s: %s", err)
			wt = 60 * time.Second
		}

		task, err = ecsService.CreateTaskWithWait(r.Context(), cluster, taskRequest, wt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		task, err = ecsService.CreateTask(r.Context(), cluster, taskRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	j, err := json.Marshal(task)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", task, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
