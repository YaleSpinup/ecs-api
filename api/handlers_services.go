package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/ecs-api/orchestration"
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/gorilla/mux"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// ServiceCreateHandler is the one stop shop for creating a service end to end with some
// basic assumptions baked into the automation
func (s *server) ServiceCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)

	log.Debugf("new service orchestration request body:\n%s", body)

	var req orchestration.ServiceOrchestrationInput
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		log.Error("cannot Decode body into create service input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	log.Debugf("decoded request into service orchestration request:\n%+v", req)

	output, err := orchestrator.CreateService(r.Context(), &req)
	if err != nil {
		log.Errorf("error in creating service orchestration: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
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

// ServiceDeleteHandler is the one stop shop for deleting a service end to end with some
// basic assumptions baked into the automation
func (s *server) ServiceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]

	// Check for the all query param
	recursive := false
	b, err := strconv.ParseBool(r.URL.Query().Get("recursive"))
	if err == nil {
		recursive = b
	}

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	output, err := orchestrator.DeleteService(r.Context(), &orchestration.ServiceDeleteInput{
		Cluster:   aws.String(cluster),
		Service:   aws.String(service),
		Recursive: recursive,
	})
	if err != nil {
		log.Errorf("error in service delete orchestration: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
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

// ServiceUpdateHandler updates a service and its dependencies
func (s *server) ServiceUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]

	body, _ := ioutil.ReadAll(r.Body)
	log.Debugf("update service (%s/%s) orchestration request body: %s", cluster, service, body)

	var req orchestration.ServiceOrchestrationUpdateInput
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		handleError(w, errors.Wrap(err, "cannot decode body into update service input"))
		return
	}

	log.Debugf("decoded request into service (%s/%s) orchestration request:\n%+v", cluster, service, req)

	orchestrator, err := s.newOrchestrator(account)
	if err != nil {
		handleError(w, err)
		return
	}

	output, err := orchestrator.UpdateService(r.Context(), cluster, service, &req)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, errors.Wrap(err, "cannot decode output into update service input"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ServiceListHandler gets a list of services in a cluster
func (s *server) ServiceListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	ecsService, ok := s.ecsServices[account]
	if !ok {
		msg := fmt.Sprintf("ecs service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	// Collect all of the task
	output, err := ecsService.ListServices(r.Context(), cluster)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the ssm service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ServiceShowHandler gets the details for a service in a cluster
func (s *server) ServiceShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	ecsService, ok := s.ecsServices[account]
	if !ok {
		msg := fmt.Sprintf("ecs service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	q := r.URL.Query()

	// Check for the all query param
	all := false
	b, err := strconv.ParseBool(q.Get("all"))
	if err == nil {
		all = b
	}

	serviceOutput, err := ecsService.GetService(r.Context(), cluster, service)
	if err != nil {
		handleError(w, err)
		return
	}

	serviceTags, err := ecsService.ListTags(r.Context(), aws.StringValue(serviceOutput.ServiceArn))
	if err != nil {
		handleError(w, err)
		return
	}

	var j []byte
	if !all {
		serviceOutput.Tags = serviceTags
		j, err = json.Marshal(serviceOutput)
		if err != nil {
			handleError(w, err)
			return
		}
	} else {
		log.Debugf("getting all details about %s/%s", cluster, service)

		cluOutput, err := ecsService.GetCluster(r.Context(), aws.String(cluster))
		if err != nil {
			handleError(w, err)
			return
		}

		tdOutput, _, err := ecsService.GetTaskDefinition(r.Context(), serviceOutput.TaskDefinition, false)
		if err != nil {
			handleError(w, err)
			return
		}

		tasks, err := ecsService.ListTasks(r.Context(), cluster, service, []string{"STOPPED", "RUNNING"})
		if err != nil {
			handleError(w, err)
			return
		}

		var serviceDiscoveryEndpoint *string
		sd, ok := s.sdServices[account]
		if ok {
			log.Debugf("found service discovery account information for all details lookup of %s/%s", cluster, service)
			serviceDiscoveryEndpoint, err = sd.ServiceEndpoint(r.Context(), aws.StringValue(serviceOutput.ServiceRegistries[0].RegistryArn))
			if err != nil {
				log.Errorf("error getting servicediscovery endpoint for %s/%s: %s", cluster, service, err)
			}
		}

		output := struct {
			*ecs.Service
			Cluster         *ecs.Cluster
			ServiceEndpoint *string
			Tasks           []*string
			TaskDefinition  *ecs.TaskDefinition
			Tags            []*ecs.Tag
		}{
			Cluster:         cluOutput,
			Service:         serviceOutput,
			ServiceEndpoint: serviceDiscoveryEndpoint,
			Tasks:           tasks,
			TaskDefinition:  tdOutput,
			Tags:            serviceTags,
		}

		j, err = json.Marshal(output)
		if err != nil {
			handleError(w, err)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s server) newOrchestrator(account string) (*orchestration.Orchestrator, error) {
	log.Debugf("creating new orchestrator for account %s", account)

	cwlService, ok := s.cwLogsServices[account]
	if !ok {
		msg := fmt.Sprintf("cloudwatchlogs service not found for account: %s", account)
		return nil, apierror.New(apierror.ErrNotFound, msg, nil)
	}

	ecsService, ok := s.ecsServices[account]
	if !ok {
		msg := fmt.Sprintf("ecs service not found for account: %s", account)
		return nil, apierror.New(apierror.ErrNotFound, msg, nil)
	}

	iamService, ok := s.iamServices[account]
	if !ok {
		msg := fmt.Sprintf("iam service not found for account: %s", account)
		return nil, apierror.New(apierror.ErrNotFound, msg, nil)
	}

	rgTaggingAPIService, ok := s.rgTaggingAPIServices[account]
	if !ok {
		msg := fmt.Sprintf("resourcegroups tagging service not found for account: %s", account)
		return nil, apierror.New(apierror.ErrNotFound, msg, nil)
	}

	sdService, ok := s.sdServices[account]
	if !ok {
		msg := fmt.Sprintf("service discovery service not found for account: %s", account)
		return nil, apierror.New(apierror.ErrNotFound, msg, nil)
	}

	smService, ok := s.smServices[account]
	if !ok {
		msg := fmt.Sprintf("secretsmanager service not found for account: %s", account)
		return nil, apierror.New(apierror.ErrNotFound, msg, nil)
	}

	return &orchestration.Orchestrator{
		CloudWatchLogs:           cwlService,
		ECS:                      ecsService,
		IAM:                      iamService,
		ResourceGroupsTaggingAPI: rgTaggingAPIService,
		SecretsManager:           smService,
		ServiceDiscovery:         sdService,
		DefaultSecurityGroups:    ecsService.DefaultSgs,
		DefaultSubnets:           ecsService.DefaultSubnets,
		DefaultPublic:            "DISABLED",
		Token:                    uuid.NewV4().String(),
		Org:                      s.org,
	}, nil
}

// ServiceEventsHandler gets the events for a service in a cluster
func (s *server) ServiceEventsHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	ecsService, ok := s.ecsServices[account]
	if !ok {
		msg := fmt.Sprintf("ecs service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	output, err := ecsService.GetService(r.Context(), cluster, service)
	if err != nil {
		handleError(w, err)
		return
	}

	events := output.Events
	j, err := json.Marshal(events)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ServiceLogsHandler gets the logs for a task/container by using the cluster name as
// the log group name and constructing the log stream from the service name, the task id, and the container name
func (s *server) ServiceLogsHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	task := vars["task"]
	container := vars["container"]

	logService, ok := s.cwLogsServices[account]
	if !ok {
		msg := fmt.Sprintf("cloudwatch logs service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	logStream := fmt.Sprintf("%s/%s/%s", service, container, task)
	log.Debugf("getting events for log group/stream: %s/%s", cluster, logStream)

	input := cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(cluster),
		LogStreamName: aws.String(logStream),
	}

	if err := parseLogQuery(r, &input); err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "failed to parse log query", err))
		return
	}

	log.Debugf("requesting log events with input: %+v", input)
	output, err := logService.GetLogEvents(r.Context(), &input)
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// parseLogQuery processes the query parameters for logs
func parseLogQuery(r *http.Request, input *cloudwatchlogs.GetLogEventsInput) error {
	for name, values := range r.URL.Query() {
		switch name {
		case "container", "task":
			log.Debugf("ignoring %s parameter", name)
		case "limit":
			limit := values[0]
			log.Debugf("processing limit value '%s'", limit)
			l, err := strconv.ParseInt(limit, 10, 64)
			if err != nil {
				return err
			}
			input.Limit = aws.Int64(l)
		case "seq":
			seq := values[0]
			log.Debugf("processing sequence value '%s'", seq)
			input.NextToken = aws.String(seq)
			input.StartFromHead = aws.Bool(true)
		case "start":
			start := values[0]
			log.Debugf("processing start value '%s'", start)
			s, err := strconv.ParseInt(start, 10, 64)
			if err != nil {
				return err
			}
			input.StartTime = aws.Int64(s)
		case "end":
			end := values[0]
			log.Debugf("processing end value '%s'", end)
			e, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				return err
			}
			input.EndTime = aws.Int64(e)
		default:
			log.Warnf("unexpected query parameter '%s'", name)
		}
	}
	return nil
}
