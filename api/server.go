package api

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/YaleSpinup/ecs-api/cloudwatchlogs"
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/YaleSpinup/ecs-api/ecs"
	"github.com/YaleSpinup/ecs-api/servicediscovery"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

type server struct {
	sdServices     map[string]servicediscovery.ServiceDiscovery
	ecsServices    map[string]ecs.ECS
	cwLogsServices map[string]cloudwatchlogs.CloudWatchLogs
	router         *mux.Router
	version        common.Version
	context        context.Context
}

// NewServer creates a new server and starts it
func NewServer(config common.Config) error {
	// setup server context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server{
		sdServices:     make(map[string]servicediscovery.ServiceDiscovery),
		ecsServices:    make(map[string]ecs.ECS),
		cwLogsServices: make(map[string]cloudwatchlogs.CloudWatchLogs),
		router:         mux.NewRouter(),
		version:        config.Version,
		context:        ctx,
	}

	for name, c := range config.Accounts {
		log.Debugf("Creating new services for account '%s' with key '%s' in region '%s'", name, c.Akid, c.Region)
		s.sdServices[name] = servicediscovery.NewSession(c)
		s.ecsServices[name] = ecs.NewSession(c)
		s.cwLogsServices[name] = cloudwatchlogs.NewSession(c)
	}

	publicURLs := map[string]string{
		"/v1/ecs/ping":    "public",
		"/v1/ecs/version": "public",
		"/v1/ecs/metrics": "public",
	}

	// load routes
	s.routes()

	if config.ListenAddress == "" {
		config.ListenAddress = ":8080"
	}
	handler := handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, TokenMiddleware(config.Token, publicURLs, s.router)))
	srv := &http.Server{
		Handler:      handler,
		Addr:         config.ListenAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof("Starting listener on %s", config.ListenAddress)
	if err := srv.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

// LogWriter is an http.ResponseWriter
type LogWriter struct {
	http.ResponseWriter
}

// Write log message if http response writer returns an error
func (w LogWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	if err != nil {
		log.Errorf("Write failed: %v", err)
	}
	return
}

// rollBack executes functions from a stack of rollback functions
func rollBack(t *[]func() error) {
	if t == nil {
		return
	}

	tasks := *t
	log.Errorf("executing rollback of %d tasks", len(tasks))
	for i := len(tasks) - 1; i >= 0; i-- {
		f := tasks[i]
		if funcerr := f(); funcerr != nil {
			log.Errorf("rollback task error: %s, continuing rollback", funcerr)
		}
	}
}
