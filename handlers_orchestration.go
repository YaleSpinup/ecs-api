package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/YaleSpinup/ecs-api/orchestration"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// ServiceOrchestrationCreateHandler is the one stop shop for creating a service
// end to end with some basic assumptions baked into the automation
func ServiceOrchestrationCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]

	ec, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sd, ok := SdServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	orchestrator := orchestration.Orchestrator{
		ECS:              ec.Service,
		ServiceDiscovery: sd.Service,
		Token:            uuid.NewV4().String(),
	}

	sgs := []*string{}
	for _, sg := range AppConfig.Accounts[account].DefaultSgs {
		sgs = append(sgs, aws.String(sg))
	}

	if len(sgs) > 0 {
		orchestration.DefaultSecurityGroups = sgs
	}

	sus := []*string{}
	for _, su := range AppConfig.Accounts[account].DefaultSubnets {
		sus = append(sus, aws.String(su))
	}

	if len(sus) > 0 {
		orchestration.DefaultSubnets = sus
	}

	if AppConfig.Accounts[account].DefaultExecutionRoleArn != "" {
		orchestration.DefaultExecutionRoleArn = aws.String(AppConfig.Accounts[account].DefaultExecutionRoleArn)
	}

	body, _ := ioutil.ReadAll(r.Body)
	log.Debugf("new service orchestration request body: %s", body)

	var req orchestration.ServiceOrchestrationInput
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		log.Error("cannot Decode body into create service input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	log.Debugf("decoded request into service orchestration request:\n%+v", req)

	output, err := orchestrator.CreateService(r.Context(), &req)
	if err != nil {
		log.Errorf("error in creating service orchestration: %s", err)
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

// ServiceOrchestrationDeleteHandler is the one stop shop for deleting a service
// end to end with some basic assumptions baked into the automation
func ServiceOrchestrationDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]

	ec, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sd, ok := SdServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	orchestrator := orchestration.Orchestrator{
		ECS:              ec.Service,
		ServiceDiscovery: sd.Service,
		Token:            uuid.NewV4().String(),
	}

	req := orchestration.ServiceDeleteInput{
		// Recursive: true,
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("cannot Decode body into delete service input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	log.Debugf("decoded request into service deleted orchestration request:\n%+v", req)

	output, err := orchestrator.DeleteService(r.Context(), &req)
	if err != nil {
		log.Errorf("error in service delete orchestration: %s", err)
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
