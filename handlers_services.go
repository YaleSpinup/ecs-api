package main

import (
	"encoding/json"
	"net/http"
	"time"

	"git.yale.edu/spinup/ecs-api/ecs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ServiceCreateHandler creates a service in a cluster
func ServiceCreateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var serviceRequest ecs.ServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&serviceRequest); err != nil {
		log.Error("Cannot Decode body into create task request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	log.Debugf("Decoded request into service request: %+v", serviceRequest)

	var service *ecs.Service
	var err error
	if wait := r.URL.Query().Get("wait"); wait != "" {
		var w time.Duration
		w, err = time.ParseDuration(wait)
		if err != nil {
			w = 60 * time.Second
		}
		service, err = ecsService.CreateServiceWithWait(r.Context(), cluster, serviceRequest, w)
	} else {
		service, err = ecsService.CreateService(r.Context(), cluster, serviceRequest)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	j, err := json.Marshal(service)
	if err != nil {
		log.Errorf("Cannot marshal response (%v) into JSON: %s", service, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ServiceListHandler gets a list of services in a cluster
func ServiceListHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	services, err := ecsService.ListServices(r.Context(), cluster)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(services)
	if err != nil {
		log.Errorf("Cannot marshal response (%v) into JSON: %s", services, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ServiceShowHandler gets the details for a service in a cluster
func ServiceShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if service == "" {
		log.Errorf("service not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := ecsService.GetService(r.Context(), cluster, service)
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

// ServiceDeleteHandler stops a service in a cluster
func ServiceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := ecsService.DeleteService(r.Context(), cluster, service)
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
