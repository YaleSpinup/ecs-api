package api

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/YaleSpinup/ecs-api/cloudwatchlogs"
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/YaleSpinup/ecs-api/ecs"
	"github.com/YaleSpinup/ecs-api/elbv2"
	"github.com/YaleSpinup/ecs-api/iam"
	"github.com/YaleSpinup/ecs-api/resourcegroupstaggingapi"
	"github.com/YaleSpinup/ecs-api/secretsmanager"
	"github.com/YaleSpinup/ecs-api/servicediscovery"
	"github.com/YaleSpinup/ecs-api/ssm"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

type server struct {
	cwLogsServices       map[string]cloudwatchlogs.CloudWatchLogs
	ecsServices          map[string]ecs.ECS
	elbv2Services        map[string]elbv2.ELBV2API
	iamServices          map[string]iam.IAM
	rgTaggingAPIServices map[string]resourcegroupstaggingapi.ResourceGroupsTaggingAPI
	sdServices           map[string]servicediscovery.ServiceDiscovery
	smServices           map[string]secretsmanager.SecretsManager
	ssmServices          map[string]ssm.SSM
	router               *mux.Router
	version              common.Version
	org                  string
}

// NewServer creates a new server and starts it
func NewServer(config common.Config) error {
	s := server{
		cwLogsServices:       make(map[string]cloudwatchlogs.CloudWatchLogs),
		ecsServices:          make(map[string]ecs.ECS),
		elbv2Services:        make(map[string]elbv2.ELBV2API),
		iamServices:          make(map[string]iam.IAM),
		rgTaggingAPIServices: make(map[string]resourcegroupstaggingapi.ResourceGroupsTaggingAPI),
		sdServices:           make(map[string]servicediscovery.ServiceDiscovery),
		smServices:           make(map[string]secretsmanager.SecretsManager),
		ssmServices:          make(map[string]ssm.SSM),
		router:               mux.NewRouter(),
		version:              config.Version,
		org:                  config.Org,
	}

	for name, c := range config.Accounts {
		log.Debugf("Creating new services for account '%s' with key '%s' in region '%s'", name, c.Akid, c.Region)
		s.cwLogsServices[name] = cloudwatchlogs.NewSession(c)
		s.ecsServices[name] = ecs.NewSession(c)
		s.elbv2Services[name] = elbv2.NewSession(c)
		s.iamServices[name] = iam.NewSession(c)
		s.rgTaggingAPIServices[name] = resourcegroupstaggingapi.NewSession(c)
		s.sdServices[name] = servicediscovery.NewSession(c)
		s.smServices[name] = secretsmanager.NewSession(c)
		s.ssmServices[name] = ssm.NewSession(c)
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
	handler := handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, TokenMiddleware([]byte(config.Token), publicURLs, s.router)))
	srv := &http.Server{
		Handler:      handler,
		Addr:         config.ListenAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof("Starting listener on %s", config.ListenAddress)
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Errorf("error starting listener: %s", err)
			os.Exit(1)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// setup server context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	log.Warn("shutting down")
	os.Exit(0)

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
