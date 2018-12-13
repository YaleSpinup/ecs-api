package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/YaleSpinup/ecs-api/cloudwatchlogs"
	"github.com/YaleSpinup/ecs-api/servicediscovery"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/YaleSpinup/ecs-api/ecs"
	"github.com/YaleSpinup/ecs-api/ecsapi"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

var (
	// Version is the main version number
	Version = ecsapi.Version

	// VersionPrerelease is a prerelease marker
	VersionPrerelease = ecsapi.VersionPrerelease

	// buildstamp is the timestamp the binary was built, it should be set at buildtime with ldflags
	buildstamp = ecsapi.BuildStamp

	// githash is the git sha of the built binary, it should be set at buildtime with ldflags
	githash = ecsapi.GitHash

	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
)

// AppConfig holds the configuration information for the app
var AppConfig common.Config

// EcsServices is a global map of ECS services
var EcsServices = make(map[string]ecs.ECS)

// SdServices is a global map of ServiceDiscovery services
var SdServices = make(map[string]servicediscovery.ServiceDiscovery)

// LogServerices is a global map of Cloudwatch Logs services
var LogServices = make(map[string]cloudwatchlogs.CloudWatchLogs)

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	log.Infof("Starting ECS-API version %s%s", Version, VersionPrerelease)

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalln("Unable to open config file", err)
	}

	r := bufio.NewReader(configFile)
	config, err := common.ReadConfig(r)
	if err != nil {
		log.Fatalf("Unable to read configuration from %s.  %+v", *configFileName, err)
	}
	AppConfig = config

	// Set the loglevel, info if it's unset
	switch AppConfig.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	log.Debugf("Read config: %+v", AppConfig)

	// Create a shared ECS session, service discovery session and cloudwatch logs session for each account
	for name, c := range AppConfig.Accounts {
		log.Debugf("Creating new ECS service for account '%s' with key '%s' in region '%s'", name, c.Akid, c.Region)
		EcsServices[name] = ecs.NewSession(c)

		log.Debugf("Creating new service discovery service for account '%s' with key '%s' in region '%s'", name, c.Akid, c.Region)
		SdServices[name] = servicediscovery.NewSession(c)

		log.Debugf("Creating new cloudwatch logs service for account '%s' with key '%s' in region '%s'", name, c.Akid, c.Region)
		LogServices[name] = cloudwatchlogs.NewSession(c)
	}

	publicURLs := map[string]string{
		"/v1/ecs/ping":    "public",
		"/v1/ecs/version": "public",
	}

	router := mux.NewRouter()
	api := router.PathPrefix("/v1/ecs").Subrouter()
	api.HandleFunc("/ping", PingHandler)
	api.HandleFunc("/version", VersionHandler)

	// Service Orchestration handlers
	api.HandleFunc("/{account}/services", ServiceOrchestrationCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/services", ServiceOrchestrationDeleteHandler).Methods(http.MethodDelete)

	// Clusters handlers
	api.HandleFunc("/{account}/clusters", ClusterListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters", ClusterCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/clusters/{cluster}", ClusterShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}", ClusterDeleteHandler).Methods(http.MethodDelete)

	// Services handlers
	api.HandleFunc("/{account}/clusters/{cluster}/services", ServiceListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services", ServiceCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}", ServiceShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}", ServiceDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/events", ServiceEventsHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/services/{service}/logs", ServiceLogsHandler).Methods(http.MethodGet).Queries("task", "{task}", "container", "{container}")

	// Tasks handlers
	api.HandleFunc("/{account}/clusters/{cluster}/tasks", TaskListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/tasks", TaskCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/clusters/{cluster}/tasks/{task}", TaskShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/clusters/{cluster}/tasks/{task}", TaskDeleteHandler).Methods(http.MethodDelete)

	// Task definitions handlers
	api.HandleFunc("/{account}/taskdefs", TaskDefListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/taskdefs", TaskDefCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/taskdefs/{taskdef}", TaskDefShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/taskdefs/{taskdef}", TaskDefDeleteHandler).Methods(http.MethodDelete)

	// Service Discovery handlers
	api.HandleFunc("/{account}/servicediscovery/services", ServiceDiscoveryServiceListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/servicediscovery/services", ServiceDiscoveryServiceCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/servicediscovery/services/{id}", ServiceDiscoveryServiceShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/servicediscovery/services/{id}", ServiceDiscoveryServiceDeleteHandler).Methods(http.MethodDelete)

	if AppConfig.ListenAddress == "" {
		AppConfig.ListenAddress = ":8080"
	}
	handler := handlers.LoggingHandler(os.Stdout, TokenMiddleware(publicURLs, router))
	srv := &http.Server{
		Handler:      handler,
		Addr:         AppConfig.ListenAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof("Starting listener on %s", AppConfig.ListenAddress)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// LogWriter is an http.ResponseWriter
type LogWriter struct {
	http.ResponseWriter
}

// Write log message if http response writer returns and error
func (w LogWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	if err != nil {
		log.Errorf("Write failed: %v", err)
	}
	return
}

func vers() {
	fmt.Printf("ECS-API Version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}
