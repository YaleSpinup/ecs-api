package main

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// TaskCreateHandler runs a task in a cluster.  It expects to marshall the
// request body into RunTaskInput.
// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#RunTaskInput
func TaskCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var taskRequest ecs.RunTaskInput
	err := json.NewDecoder(r.Body).Decode(&taskRequest)
	if err != nil {
		log.Error("cannot Decode body into run task input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	taskRequest.Cluster = aws.String(cluster)
	taskRequest.Cluster = aws.String("FARGATE")

	log.Debugf("decoded request into task request: %+v", taskRequest)

	output, err := ecsService.Service.RunTaskWithContext(r.Context(), &taskRequest)
	if err != nil {
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

// TaskListHandler gets a list of tasks in a cluster
func TaskListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account or cluster not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	service := q.Get("service")
	family := q.Get("family")
	status := q.Get("status")

	input := ecs.ListTasksInput{
		Cluster:    aws.String(cluster),
		LaunchType: aws.String("FARGATE"),
	}

	if family != "" {
		log.Infof("filtering task response by family %s", family)
		input.Family = aws.String(family)
	}

	if service != "" {
		log.Infof("filtering task response by service %s", service)
		input.ServiceName = aws.String(service)
	}

	if status != "" {
		log.Infof("filtering task response by desired status %s", status)
		input.DesiredStatus = aws.String(status)
	}

	// Collect all of the task
	output := []string{}
	for {
		out, err := ecsService.Service.ListTasksWithContext(r.Context(), &input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		for _, t := range out.TaskArns {
			output = append(output, aws.StringValue(t))
		}

		if out.NextToken == nil {
			break
		}
		input.NextToken = out.NextToken
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

// TaskShowHandler gets the details for a task in a cluster
func TaskShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
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

	output, err := ecsService.Service.DescribeTasksWithContext(r.Context(), &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   aws.StringSlice([]string{task}),
	})
	if err != nil {
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

// TaskDeleteHandler stops a task in a cluster
func TaskDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
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

	// TODO: support reason?  Reason: aws.String("because foobar")
	output, err := ecsService.Service.StopTaskWithContext(r.Context(), &ecs.StopTaskInput{
		Cluster: aws.String(cluster),
		Task:    aws.String(task),
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		log.Errorf("Cannot marshal response (%v) into JSON: %s", output, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
