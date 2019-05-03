package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ClusterCreateHandler creates a new cluster
func ClusterCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req ecs.CreateClusterInput
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error("cannot decode body into create cluster request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cluster, err := ecsService.Service.CreateClusterWithContext(r.Context(), &req)
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
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Collect all of the task
	input := ecs.ListClustersInput{}
	output := []string{}
	for {
		out, err := ecsService.Service.ListClustersWithContext(r.Context(), &input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		for _, t := range out.ClusterArns {
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

// ClusterShowHandler gets details about a cluster
func ClusterShowHandler(w http.ResponseWriter, r *http.Request) {
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

	log.Infof("getting cluster %s details", cluster)
	resp, err := ecsService.Service.DescribeClustersWithContext(r.Context(), &ecs.DescribeClustersInput{
		Clusters: aws.StringSlice([]string{cluster}),
	})

	if err != nil {
		log.Errorf("error describing cluster: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	log.Debugf("get cluster output: %+v", resp)

	if len(resp.Clusters) != 1 {
		log.Errorf("unexpected cluster response (length: %d): %v", len(resp.Clusters), resp.Clusters)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("unexpected cluster response (length != 1: %d)", len(resp.Clusters))))
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

	resp, err := ecsService.Service.DeleteClusterWithContext(r.Context(), &ecs.DeleteClusterInput{
		Cluster: aws.String(cluster),
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
