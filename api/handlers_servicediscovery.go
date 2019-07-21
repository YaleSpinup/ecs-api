package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/ecs-api/apierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ServiceDiscoveryServiceListHandler gets the list of service discovery services
func (s *server) ServiceDiscoveryServiceListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]

	sd, ok := s.sdServices[account]
	if !ok {
		msg := fmt.Sprintf("service discover service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	resp, err := sd.Service.ListServicesWithContext(r.Context(), &servicediscovery.ListServicesInput{})
	if err != nil {
		log.Errorf("error listing servicediscovery services: %s", err)
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

// ServiceDiscoveryServiceShowHandler gets the details for a service discovery service from an ID
func (s *server) ServiceDiscoveryServiceShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	id := vars["id"]

	sd, ok := s.sdServices[account]
	if !ok {
		msg := fmt.Sprintf("service discover service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	resp, err := sd.Service.GetServiceWithContext(r.Context(), &servicediscovery.GetServiceInput{
		Id: aws.String(id),
	})

	if err != nil {
		log.Errorf("error describing servicediscovery service: %s", err)
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

// ServiceDiscoveryServiceDeleteHandler deletes a service discovery service by ID
func (s *server) ServiceDiscoveryServiceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	id := vars["id"]

	sd, ok := s.sdServices[account]
	if !ok {
		msg := fmt.Sprintf("service discover service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	resp, err := sd.Service.DeleteServiceWithContext(r.Context(), &servicediscovery.DeleteServiceInput{
		Id: aws.String(id),
	})

	if err != nil {
		log.Errorf("error deleting servicediscovery service: %s", err)
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

// ServiceDiscoveryServiceCreateHandler creates a service discovery service
//
// Expects input JSON to satisfy serviceDiscovery.CreateServiceInput{}
// https://docs.aws.amazon.com/sdk-for-go/api/service/servicediscovery/#CreateServiceInput
func (s *server) ServiceDiscoveryServiceCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]

	sd, ok := s.sdServices[account]
	if !ok {
		msg := fmt.Sprintf("service discover service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	var req servicediscovery.CreateServiceInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Cannot Decode body into create servicediscover service input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	log.Debugf("Decoded request into service request: %+v", req)

	resp, err := sd.Service.CreateServiceWithContext(r.Context(), &req)
	if err != nil {
		log.Errorf("error describing servicediscovery service: %s", err)
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