package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ServiceCreateHandler creates a service in a cluster
func ServiceCreateHandler(w http.ResponseWriter, r *http.Request) {
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

	var serviceRequest ecs.CreateServiceInput
	if err := json.NewDecoder(r.Body).Decode(&serviceRequest); err != nil {
		log.Error("cannot Decode body into create task request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	serviceRequest.Cluster = aws.String(cluster)

	log.Debugf("decoded request into service request: %+v", serviceRequest)

	output, err := ecsService.Service.CreateServiceWithContext(r.Context(), &serviceRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
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
func ServiceListHandler(w http.ResponseWriter, r *http.Request) {
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
func ServiceShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
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
		sd, ok := SdServices[account]
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
func ServiceEventsHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
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

// ServiceDeleteHandler stops a service in a cluster
func ServiceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	ecsService, ok := EcsServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	output, err := ecsService.Service.DeleteServiceWithContext(r.Context(), &ecs.DeleteServiceInput{
		Cluster: aws.String(cluster),
		Force:   aws.Bool(true),
		Service: aws.String(service),
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


// ServiceLogsHandler gets the logs for a task/container by using the cluster name as
// the log group name and constructing the log stream from the service name, the task id, and the container name 
func ServiceLogsHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	service := vars["service"]
	task := vars["task"]
	container := vars["container"]

	logService, ok := LogServices[account]
	if !ok {
		log.Errorf("account not found: %s", account)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logStream := fmt.Sprintf("%s/%s/%s", service, container,task)
	log.Debugf("getting events for log group/stream: %s/%s", cluster, logStream)

	output, err := logService.Service.GetLogEventsWithContext(r.Context(), &cloudwatchlogs.GetLogEventsInput{
		LogGroupName: aws.String(cluster),
		LogStreamName:   aws.String(logStream),
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

