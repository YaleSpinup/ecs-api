package main

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// TaskDefCreateHandler creates a task definition. The expected input is compatible with
// the AWS SDK ResgisterTaskDefinitionInput struct
// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#RegisterTaskDefinitionInput
func TaskDefCreateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tdefInput := ecs.RegisterTaskDefinitionInput{}
	err := json.NewDecoder(r.Body).Decode(&tdefInput)
	if err != nil {
		log.Error("cannot decode body into create taskdef request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	resp, err := ecsService.Service.RegisterTaskDefinitionWithContext(r.Context(), &tdefInput)
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

// TaskDefListHandler returns a list of task definitions
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

	// allow for a prefix query parameter
	var prefix *string
	if q.Get("prefix") != "" {
		prefix = aws.String(q.Get("prefix"))
	}

	// Collect all of the task definitions and versions for now
	input := ecs.ListTaskDefinitionsInput{Status: aws.String(status), FamilyPrefix: prefix}
	output := []string{}
	for {
		out, err := ecsService.Service.ListTaskDefinitionsWithContext(r.Context(), &input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		for _, td := range out.TaskDefinitionArns {
			output = append(output, aws.StringValue(td))
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

// TaskDefShowHandler gets the details for a task definition
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

	resp, err := ecsService.Service.DescribeTaskDefinitionWithContext(r.Context(), &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskdef),
	})
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

	resp, err := ecsService.Service.DeregisterTaskDefinitionWithContext(r.Context(), &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String(taskdef),
	})
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
