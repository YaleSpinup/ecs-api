package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"git.yale.edu/spinup/ecs-api/common"
	"git.yale.edu/spinup/ecs-api/ecs"
	"git.yale.edu/spinup/ecs-api/ecsapi"
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
	buildstamp = "No BuildStamp Provided"

	// githash is the git sha of the built binary, it should be set at buildtime with ldflags
	githash = "No Git Commit Provided"

	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
)

// AppConfig holds the configuration information for the app
var AppConfig common.Config

// EcsServices is a global map of ECS services
var EcsServices = make(map[string]ecs.ECS)

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

	// Create a shared ECS session for each account
	for name, c := range AppConfig.Accounts {
		log.Debugf("Creating new ECS service for account '%s' with key '%s' in region '%s'", name, c.Akid, c.Region)
		EcsServices[name] = ecs.NewSession(c)
	}

	publicURLs := map[string]string{
		"/v1/ecs/ping":    "public",
		"/v1/ecs/version": "public",
	}

	router := mux.NewRouter()
	api := router.PathPrefix("/v1/ecs").Subrouter()
	api.HandleFunc("/ping", PingHandler)
	api.HandleFunc("/version", VersionHandler)

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

func vers() {
	fmt.Printf("Indexer Version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}
