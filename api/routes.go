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

	// Docker image handlers
	api.HandleFunc("/images", s.ImageVerificationHandler).Methods(http.MethodHead).Queries("image", "{image}")

	// Service handlers
	api.HandleFunc("/{account}/services", s.ServiceCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/clusters/{cluster}/services", s.ServiceListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}", s.ServiceUpdateHandler).Methods(http.MethodPut)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}", s.ServiceDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}", s.ServiceShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/events", s.ServiceEventsHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/logs", s.ServiceLogsHandler).Methods(http.MethodGet).
		Queries("task", "{task}", "container", "{container}")
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/logs", s.ServiceLogsHandler).Methods(http.MethodGet).
		Queries("task", "{task}", "container", "{container}", "limit", "{limit}", "seq", "{seq}")
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/logs", s.ServiceLogsHandler).Methods(http.MethodGet).
		Queries("task", "{task}", "container", "{container}", "start", "{start}", "end", "{end}")
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/logs", s.ServiceLogsHandler).Methods(http.MethodGet).
		Queries("task", "{task}", "container", "{container}", "start", "{start}", "end", "{end}", "limit", "{limit}", "seq", "{seq}")

	// Tasks handlers
	api.HandleFunc("/{account}/clusters/{cluster}/tasks/{task}", s.TaskShowHandler).Methods(http.MethodGet)

	// Secrets handlers
	api.HandleFunc("/{account}/secrets", s.SecretListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/secrets", s.SecretCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/secrets/{secret}", s.SecretShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/secrets/{secret}", s.SecretDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/secrets/{secret}", s.SecretUpdateHandler).Methods(http.MethodPut)

	// Parameter store handlers
	api.HandleFunc("/{account}/params/{prefix}", s.ParamCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/params/{prefix}", s.ParamListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/params/{prefix}", s.ParamDeleteAllHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/params/{prefix}/{param}", s.ParamShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/params/{prefix}/{param}", s.ParamDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/params/{prefix}/{param}", s.ParamUpdateHandler).Methods(http.MethodPut)
}
