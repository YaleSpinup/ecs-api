package main

import (
	"encoding/json"
	"net/http"

	"git.yale.edu/spinup/ecs-api/ecs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ClusterCreateHandler creates a new cluster
func ClusterCreateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req ecs.ClusterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error("cannot decode body into create cluster request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cluster, err := ecsService.CreateCluster(r.Context(), req.Name)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(cluster)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", cluster, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ClusterListHandler gets a list of clusters
func ClusterListHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clusters, err := ecsService.ListClusters(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	j, err := json.Marshal(clusters)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", clusters, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ClusterShowHandler gets details about a cluster
func ClusterShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := ecsService.GetCluster(r.Context(), cluster)
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

// ClusterDeleteHandler deletes cluster
func ClusterDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := ecsService.DeleteCluster(r.Context(), cluster)
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
