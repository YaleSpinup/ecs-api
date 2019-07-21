package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *server) routes() {
	api := s.router.PathPrefix("/v1/ecs").Subrouter()
	api.HandleFunc("/ping", s.PingHandler)
	api.HandleFunc("/version", s.VersionHandler)
	api.Handle("/metrics", promhttp.Handler())

	// Service Orchestration handlers
	api.HandleFunc("/{account}/services", s.ServiceOrchestrationCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/services", s.ServiceOrchestrationUpdateHandler).Methods(http.MethodPut)
	api.HandleFunc("/{account}/services", s.ServiceOrchestrationDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/services/{service}", s.ServiceOrchestrationShowHandler).Methods(http.MethodGet)

	// Clusters handlers
	api.HandleFunc("/{account}/clusters", s.ClusterListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters", s.ClusterCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/clusters/{cluster}", s.ClusterShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}", s.ClusterDeleteHandler).Methods(http.MethodDelete)

	// Services handlers
	api.HandleFunc("/{account}/clusters/{cluster}/services", s.ServiceListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services", s.ServiceCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}", s.ServiceShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}", s.ServiceDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/events", s.ServiceEventsHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/logs", s.ServiceLogsHandler).Methods(http.MethodGet).Queries("task", "{task}", "container", "{container}")

	// Tasks handlers
	api.HandleFunc("/{account}/clusters/{cluster}/tasks", s.TaskListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/tasks", s.TaskCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/clusters/{cluster}/tasks/{task}", s.TaskShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/tasks/{task}", s.TaskDeleteHandler).Methods(http.MethodDelete)

	// Task definitions handlers
	api.HandleFunc("/{account}/taskdefs", s.TaskDefListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/taskdefs", s.TaskDefCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/taskdefs/{taskdef}", s.TaskDefShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/taskdefs/{taskdef}", s.TaskDefDeleteHandler).Methods(http.MethodDelete)

	// Service Discovery handlers
	api.HandleFunc("/{account}/servicediscovery/services", s.ServiceDiscoveryServiceListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/servicediscovery/services", s.ServiceDiscoveryServiceCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/servicediscovery/services/{id}", s.ServiceDiscoveryServiceShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/servicediscovery/services/{id}", s.ServiceDiscoveryServiceDeleteHandler).Methods(http.MethodDelete)

	// Secrets handlers
	api.HandleFunc("/{account}/secrets", s.SecretListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/secrets", s.SecretCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/secrets/{secret}", s.SecretShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/secrets/{secret}", s.SecretDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/secrets/{secret}", s.SecretUpdateHandler).Methods(http.MethodPut)
}