package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/YaleSpinup/ecs-api/orchestration"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/servicediscovery"

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

	sgs := []*string{}
	for _, sg := range orchestrator.ECS.DefaultSgs {
		sgs = append(sgs, aws.String(sg))
	}

	if len(sgs) > 0 {
		orchestration.DefaultSecurityGroups = sgs
	}

	sus := []*string{}
	for _, su := range orchestrator.ECS.DefaultSubnets {
		sus = append(sus, aws.String(su))
	}

	if len(sus) > 0 {
		orchestration.DefaultSubnets = sus
	}

	body, _ := ioutil.ReadAll(r.Body)
	log.Debugf("new service orchestration request body: %s", body)

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

	if cluster == "" {
		handleError(w, apierror.New(apierror.ErrNotFound, "cluster cannot be empty", nil))
		return
	}

	if service == "" {
		handleError(w, apierror.New(apierror.ErrNotFound, "service cannot be empty", nil))
		return
	}

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
		log.Error("cannot Decode body into update service input")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
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
		log.Errorf("error in service update orchestration: %s", err)
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
	input := ecs.ListServicesInput{
		Cluster:    aws.String(cluster),
		LaunchType: aws.String("FARGATE"),
	}
	output := []string{}
	for {
		out, err := ecsService.Service.ListServicesWithContext(r.Context(), &input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		for _, t := range out.ServiceArns {
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

	serviceOutput, err := ecsService.Service.DescribeServicesWithContext(r.Context(), &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: aws.StringSlice([]string{service}),
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if len(serviceOutput.Services) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
		return
	}

	var j []byte
	if !all {
		j, err = json.Marshal(serviceOutput)
		if err != nil {
			log.Errorf("cannot marshal response (%v) into JSON: %s", serviceOutput, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		log.Debugf("getting all details about %s/%s", cluster, service)
		tdOutput, err := ecsService.Service.DescribeTaskDefinitionWithContext(r.Context(), &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: serviceOutput.Services[0].TaskDefinition,
		})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		tasks, err := tasksList(r.Context(), ecsService.Service, cluster, service)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		var serviceDiscoveryEndpoint *string
		sd, ok := s.sdServices[account]
		if ok {
			log.Debugf("found service discovery account information for all details lookup of %s/%s", cluster, service)
			serviceDiscoveryEndpoint, err = serviceEndpoint(r.Context(), sd.Service, serviceOutput.Services[0].ServiceRegistries[0])
			if err != nil {
				log.Errorf("error getting servicediscovery endpoint for %s/%s: %s", cluster, service, err)
			}
		}

		output := struct {
			*ecs.Service
			ServiceEndpoint *string
			Tasks           []*string
			TaskDefinition  *ecs.TaskDefinition
		}{
			Service:         serviceOutput.Services[0],
			ServiceEndpoint: serviceDiscoveryEndpoint,
			Tasks:           tasks,
			TaskDefinition:  tdOutput.TaskDefinition,
		}

		j, err = json.Marshal(output)
		if err != nil {
			log.Errorf("cannot marshal response (%v) into JSON: %s", output, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s server) newOrchestrator(account string) (*orchestration.Orchestrator, error) {
	log.Debugf("creating new orchestrator for account %s", account)

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
		ECS:              ecsService,
		IAM:              iamService,
		SecretsManager:   smService,
		ServiceDiscovery: sdService,
		Token:            uuid.NewV4().String(),
		Org:              s.org,
	}, nil
}

// tasksList collects all of the task ids (with the disired state of both running and stopped) for a service
func tasksList(ctx context.Context, es *ecs.ECS, cluster, service string) ([]*string, error) {
	tasks := []*string{}
	runningTaskOutput, err := es.ListTasksWithContext(ctx, &ecs.ListTasksInput{
		Cluster:     aws.String(cluster),
		ServiceName: aws.String(service),
		LaunchType:  aws.String("FARGATE"),
	})
	if err != nil {
		return tasks, err
	}

	stoppedTaskOutput, err := es.ListTasksWithContext(ctx, &ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		ServiceName:   aws.String(service),
		LaunchType:    aws.String("FARGATE"),
		DesiredStatus: aws.String("STOPPED"),
	})
	if err != nil {
		return tasks, err
	}

	for _, t := range append(runningTaskOutput.TaskArns, stoppedTaskOutput.TaskArns...) {
		taskArn, err := arn.Parse(aws.StringValue(t))
		if err != nil {
			return tasks, err
		}

		// task resource is the form task/xxxxxxxxxxxxx
		r := strings.SplitN(taskArn.Resource, "/", 2)
		tasks = append(tasks, aws.String(r[1]))
	}

	return tasks, nil
}

// serviceEndpoint takes the service discovery client and the ecs service registry configuration.  It first gets the
// details of the service registry given and from that determines the namespace ID.  The endpoint string is determined by
// combining the service registry service name(hostname) and the namespace name (domain name).
func serviceEndpoint(ctx context.Context, sd *servicediscovery.ServiceDiscovery, registry *ecs.ServiceRegistry) (*string, error) {
	serviceResistryArn, err := arn.Parse(aws.StringValue(registry.RegistryArn))
	if err != nil {
		log.Errorf("error parsing servicediscovery service ARN %s", err)
		return nil, err
	}

	if serviceResistryArn.Resource != "" {
		log.Debugf("getting service registry service with id %s", serviceResistryArn.Resource)

		// serviceRegistryArn.Resource is of the format 'service/srv-xxxxxxxxxxxxx', but GetServiceInput needs just the ID
		serviceID := strings.SplitN(serviceResistryArn.Resource, "/", 2)
		sdOutput, err := sd.GetServiceWithContext(ctx, &servicediscovery.GetServiceInput{
			Id: aws.String(serviceID[1]),
		})

		if err != nil {
			log.Errorf("error getting service from ID %s: %s", serviceID[1], err)
			return nil, err
		}

		if nsID := aws.StringValue(sdOutput.Service.DnsConfig.NamespaceId); nsID != "" {
			namespaceOutput, err := sd.GetNamespaceWithContext(ctx, &servicediscovery.GetNamespaceInput{
				Id: aws.String(nsID),
			})
			if err != nil {
				log.Errorf("error getting namespace %s", err)
				return nil, err
			}
			endpoint := fmt.Sprintf("%s.%s", aws.StringValue(sdOutput.Service.Name), aws.StringValue(namespaceOutput.Namespace.Name))
			return &endpoint, nil
		}
	}

	log.Warnf("service discovery endpoint not found")
	return nil, nil
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

	output, err := ecsService.Service.DescribeServicesWithContext(r.Context(), &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: aws.StringSlice([]string{service}),
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if len(output.Services) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
		return
	}

	events := output.Services[0].Events
	j, err := json.Marshal(events)
	if err != nil {
		log.Errorf("cannot marshal response (%v) into JSON: %s", events, err)
		w.WriteHeader(http.StatusInternalServerError)
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

	output, err := logService.Service.GetLogEventsWithContext(r.Context(), &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(cluster),
		LogStreamName: aws.String(logStream),
	})
	if err != nil {
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
