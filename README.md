# ECS-API

This API provides simple restful API access to Amazon's ECS Fargate service.

## Table of Contents

- [ECS-API](#ecs-api)
  - [Table of Contents](#table-of-contents)
  - [Endpoint Summary](#endpoint-summary)
  - [Definition](#definition)
    - [Clusters](#clusters)
    - [Services](#services)
    - [Tasks](#tasks)
    - [Task Definitions](#task-definitions)
    - [SSM Parameters](#ssm-parameters)
  - [Docker Image verification](#docker-image-verification)
    - [Check if an image is available](#check-if-an-image-is-available)
      - [Response](#response)
  - [Service Orchestration](#service-orchestration)
    - [Orchestrate a service update](#orchestrate-a-service-update)
      - [Request](#request)
        - [Update the tags for an existing service and force a redeployment](#update-the-tags-for-an-existing-service-and-force-a-redeployment)
        - [Update the task definition and redeploy an existing service](#update-the-task-definition-and-redeploy-an-existing-service)
        - [Update the service replica count and capacity provider strategy](#update-the-service-replica-count-and-capacity-provider-strategy)
        - [Add a container definition with credentials](#add-a-container-definition-with-credentials)
        - [Update a container definitions credentials](#update-a-container-definitions-credentials)
      - [Response](#response-1)
    - [Orchestrate a service delete](#orchestrate-a-service-delete)
      - [Request](#request-1)
      - [Response](#response-2)
    - [Get logs for a task](#get-logs-for-a-task)
      - [Request](#request-2)
        - [Examples](#examples)
  - [Managed Task Definitions](#managed-task-definitions)
    - [Create a managed task definition](#create-a-managed-task-definition)
      - [Request](#request-3)
      - [Response](#response-3)
    - [Delete a managed task definition](#delete-a-managed-task-definition)
      - [Request](#request-4)
      - [Response](#response-4)
    - [List managed task definitions in a cluster](#list-managed-task-definitions-in-a-cluster)
      - [Request](#request-5)
      - [Response](#response-5)
    - [Get a managed task definitions in a cluster](#get-a-managed-task-definitions-in-a-cluster)
      - [Request](#request-6)
      - [Response](#response-6)
    - [Run a managed task definition in a cluster](#run-a-managed-task-definition-in-a-cluster)
      - [Request](#request-7)
      - [Response](#response-7)
    - [Get a list of task definition tasks](#get-a-list-of-task-definition-tasks)
      - [Request](#request-8)
      - [Response](#response-8)
    - [Get a details of task definition task](#get-a-details-of-task-definition-task)
      - [Request](#request-9)
      - [Response](#response-9)
    - [Stop a task definition task](#stop-a-task-definition-task)
      - [Request](#request-10)
      - [Response](#response-10)
  - [SSM Parameters](#ssm-parameters-1)
    - [Create a param](#create-a-param)
      - [Request](#request-11)
      - [Response](#response-11)
    - [List parameters](#list-parameters)
      - [Response](#response-12)
    - [Show a parameter](#show-a-parameter)
      - [Response](#response-13)
    - [Delete a parameter](#delete-a-parameter)
      - [Response](#response-14)
    - [Delete all parameters in a prefix](#delete-all-parameters-in-a-prefix)
      - [Response](#response-15)
    - [Update a parameter](#update-a-parameter)
      - [Request](#request-12)
      - [Response](#response-16)
  - [Secrets](#secrets)
    - [Create a secret](#create-a-secret)
      - [Request](#request-13)
      - [Response](#response-17)
    - [List secrets](#list-secrets)
      - [Response](#response-18)
    - [Show a secret](#show-a-secret)
      - [Response](#response-19)
    - [Delete a secret](#delete-a-secret)
      - [Response](#response-20)
    - [Update a secret](#update-a-secret)
      - [Request](#request-14)
      - [Response](#response-21)
    - [List load balancers (target groups) for a space](#list-load-balancers-target-groups-for-a-space)
      - [Response](#response-22)
  - [Development](#development)
  - [Author](#author)
  - [License](#license)

## Endpoint Summary

```text
GET /v1/ecs/ping
GET /v1/ecs/version

// Docker Image handlers
HEAD /v1/ecs/images?image={image}

// Service handlers
POST /v1/ecs/{account}/services
GET /v1/ecs/{account}/clusters/{cluster}/services[?all=true]
PUT /v1/ecs/{account}/clusters/{cluster}/services
DELETE /v1/ecs/{account}/clusters/{cluster}/services/{service}[?recursive=true]
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/events

// Log handlers
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="{task}"&container="{container}[&limit={limit}][&seq={seq}][&start={start}&end={end}]"

// Tasks handlers
GET /v1/ecs/{account}/clusters/{cluster}/tasks/{task}
DELETE /v1/ecs/{account}/clusters/{cluster}/tasks/{task}

// TaskDef handlers
POST /v1/ecs/{account}/taskdefs
GET /v1/ecs/{account}/clusters/{cluster}/taskdefs
DELETE /v1/ecs/{account}/clusters/{cluster}/taskdefs/{taskdef}[?recursive=true][&force=true]
GET /v1/ecs/{account}/clusters/{cluster}/taskdefs/{taskdef}
POST /v1/ecs/{account}/clusters/{cluster}/taskdefs/{taskdef}/tasks
GET /v1/ecs/{account}/clusters/{cluster}/taskdefs/{taskdef}/tasks
GET /v1/ecs/{account}/clusters/{cluster}/taskdefs/{taskdef}/tasks/{task}
DELETE /v1/ecs/{account}/clusters/{cluster}/taskdefs/{taskdef}/tasks/{task}[?reason={reason}]

// Secrets handlers
GET /v1/ecs/{account}/secrets
POST /v1/ecs/{account}/secrets
GET /v1/ecs/{account}/secrets/{secret}
PUT /v1/ecs/{account}/secrets/{secret}
DELETE /v1/ecs/{account}/secrets/{secret}

// Parameter store handlers
POST /v1/ecs/{account}/params/{prefix}
GET /v1/ecs/{account}/params/{prefix}
DELETE /v1/ecs/{account}/params/{prefix}
GET /v1/ecs/{account}/params/{prefix}/{param}
DELETE /v1/ecs/{account}/params/{prefix}/{param}
PUT /v1/ecs//{account}/params/{prefix}/{param}

// Load balancer handlers
GET /v1/ecs/{account}/lbs?space={space}
```

## Definition

### Clusters

`Clusters` provide groupings of containers.  Clusters control the default capacity provider strategy which determines when to leverage spot and on demand containers.

### Services

`Services` are long lived/scalable tasks launched based on a task definition.  Services can be comprised of multiple `containers` (because multiple containers can be defined in a task definition!).  AWS will restart containers belonging to services when they exit or die.  `Services` can be scaled and can be associated with load balancers.  Tasks that make up a service can have IP addresses, but those addresses change when tasks are restarted.

### Tasks

`Tasks`, in contrast to `Services`, should be considered short lived.  They will not automatically be restarted or scaled by AWS and cannot be associated with a load balancer.  `Tasks`, as with `services`, can be made up of multiple containers.  Tasks can have IP addresses, but those addresses change when tasks are restarted.

### Task Definitions

An ECS `Task definition` describes a set of containers used in a `service` or `task`.  The resources managed by the `taskdefs` endpoints in this API manage task definition lifecycle in addition to dependent resources for a task (clusters, tags, etc).

### SSM Parameters

`Parameters` store string data in AWS SSM parameter store. By default, parameters are encrypted (in AWS) by the `defaultKmsKeyId` given for each `account`.

## Docker Image verification

Image verification checks if an image is available.  Currently, this endpoint is a simple `HEAD` request.  Successful http response
codes mean the image is available.

### Check if an image is available

HEAD `/v1/ecs/images?image={image}`

Passing the `X-Registry-Auth` header with a base64 encoded JSON payload will attempt to authenticate to the registry using the passed credentials.

The JSON payload should be of the form:

```json
{"username": "foouser", "password": "foopass"}
```

*Note:* Header values are **not** URL encoded.

#### Response

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | image is available                    |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | image wasn't found (or requires auth) |
| **500 Internal Server Error** | a server error occurred               |

## Service Orchestration

The service orchestration endpoints for creating and deleting services allow building and destroying services with one call to the API.

The endpoints are wrapped versions of the ECS, IAM, Secrets manager and ServiceDiscovery services from AWS.  The endpoint will determine
what has been provided and try to take the most logical action.  For example, if you provide `CreateClusterInput`, `RegisterTaskDefinitionInput`
and `CreateServiceInput`, the API will attempt to create the cluster, then the task definition and then the service using the created
resources.  If you only provide the `CreateServiceInput` with the task definition name, the cluster name and the service registries, it
will assume those resources already exist and just try to create the service.

Example request body of new cluster, new task definition, new service registry, repository credentials and the new service:

```json
{
    "cluster": {
        "clustername": "myclu"
    },
    "taskdefinition": {
        "family": "webservers",
        "cpu": "256",
        "memory": "512",
        "containerdefinitions": [
            {
                "environment": [{
                    "name": "API_HOST",
                    "value": "localhost"
                  },{
                    "name": "API_PORT",
                    "value": "1234"
                  }],
                "name": "webserver",
                "image": "nginx:alpine",
                "ports": [80,443],
                "logConfiguration": {
                  "logDriver": "awslogs",
                  "options": {
                    "awslogs-group": "myclu",
                    "awslogs-stream-prefix": "www",
                    "awslogs-region": "us-east-1",
                    "awslogs-create-group": "true"
                  }
                },
                "PortMappings": [{
                    "ContainerPort": 80,
                    "protocol": "tcp"
                  }],
                "secrets": []
            }
        ]
    },
    "service": {
        "desiredcount": 1,
        "servicename": "www"
    },
    "serviceregistry": {
      "name": "www",
      "cluster": "myclu",
      "dnsconfig": {
        "namespaceid": "ns-y3uaw6neshhbev3f",
        "dnsrecords": [
          {
            "ttl": 30,
            "type": "A"
          }
        ]
      }
    },
    "credentials": {
        "webserver": {
            "Name": "myapp-webserver-cred",
            "SecretString": "{\"username\" : \"supahman\",\"password\" : \"dontkryptonitemebro\"}",
            "Description": "myapp-webserver-cred"
        }
    },
    "tags": [
        {
            "Key": "CreatedBy",
            "Value": "netid"
        },
        {
            "Key": "OS",
            "Value": "container"
        }
    ]
}
```

Example request body of new service with existing resources:

```json
{
    "service": {
        "cluster": "myclu",
        "desiredcount": 1,
        "servicename": "webapp",
        "taskdefinition": "mytaskdef:1",
        "serviceregistries": [
            {
                "registryarn": "arn:aws:servicediscovery:us-east-1:001122334455:service/srv-tvtbgvkkxtts3qlf"
            }
        ]
    }
}
```

### Orchestrate a service update

Service update orchestration currently supports:

- updating tags
- forcing a redeployment without changing the service
- updating the task definition and redeploying
- updating the service parameters (like replica count, or capacity provider strategy)

#### Request

PUT `/v1/ecs/{account}/clusters/{cluster}/services/{service}`

##### Update the tags for an existing service and force a redeployment

```json
{
    "ForceNewDeployment": true,
    "Tags": [
        {"Key": "MyKey", "Value": "MyValue"},
        {"Key": "Application", "Value": "someprefix"},
    ]
}
```

##### Update the task definition and redeploy an existing service

```json
{
    "TaskDefinition": {
        "family": "supercool-service",
        "cpu": "256",
        "memory": "1024",
        "containerdefinitions": [
            {
                "environment": [{
                    "name": "API_HOST",
                    "value": "localhost"
                  },{
                    "name": "API_PORT",
                    "value": "1234"
                  }],
                "name": "webserver",
                "image": "nginx:alpine",
                "ports": [80,443]
            }
        ]
    },
    "ForceRedeploy": true
}
```

##### Update the service replica count and capacity provider strategy

```json
{
    "Service": {
        "DesiredCount": 1,
        "CapacityProviderStrategy": [
            {
                "CapacityProvider": "FARGATE_SPOT",
                "Base": 1,
                "Weight": 1
            }
        ]
    }
}
```

##### Add a container definition with credentials

The example below adds a `privateapi` container definition with the associated *new* credentials, or adds credentials to an existing `privateapi`
container definition.  If repository credentials are passed with a container definition (see the next example), the credentials will either be
kept and continue to be associated with the container definition (if no associated credentials are passed in the `credentials` map) or the value of
the secret with that ARN will be *overwritten* by the credentials passed in the `credentials` map if they exist!

```json
{
    "TaskDefinition": {
        "family": "supercool-service",
        "cpu": "256",
        "memory": "1024",
        "containerdefinitions": [
            {
                "name": "webserver",
                "image": "nginx:alpine"
            },
            {
                "name": "privateapi",
                "image": "myorg/privateapi:latest"
            }
        ]
    },
    "credentials": {
        "privateapi": {
            "Name": "privateapi-cred-1",
            "SecretString": "{\"username\" : \"myorguser\",\"password\" : \"super-sekret-password-string\"}",
            "Description": "privateapi-creds"
        }
    },
    "ForceRedeploy": true
}
```

##### Update a container definitions credentials

In this example, the `privateapi` container definition exists and we want to update the credentials.  By passing the repository credentials ARN
along with the container definition, this instructs the API to attempt to update that secret from the passed `credentials` map.  If no associated
credential exists in the `credentials` map, the container definition will be updated to use the passed credentials ARN.

```json
{
    "TaskDefinition": {
        "family": "supercool-service",
        "cpu": "256",
        "memory": "1024",
        "containerdefinitions": [
            {
                "name": "webserver",
                "image": "nginx:alpine"
            },
            {
                "name": "privateapi",
                "image": "myorg/privateapi:latest",
                "repositorycredentials": {
                    "credentialsparameter": "arn:aws:secretsmanager:us-east-1:001122334455:secret:privateapi-cred-1-Ol7mhU"
                }
            }
        ]
    },
    "credentials": {
        "privateapi": {
            "SecretString": "{\"username\" : \"myorguser\",\"password\" : \"new-super-sekret-password-string\"}",
        }
    },
    "ForceRedeploy": true
}
```

#### Response

The response is the service body.

```json
TODO
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **200 OK**                    | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster or service wasn't found |
| **500 Internal Server Error** | a server error occurred                  |

### Orchestrate a service delete

Service delete orchestration supports deleting a service or recursively deleting a service and its dependencies.

#### Request

DELETE `/v1/ecs/{account}/clusters/{cluster}/services/{service}[?recursive=true]`

#### Response

The response is the service body.

```json
TODO
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **200 OK**                    | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster or service wasn't found |
| **500 Internal Server Error** | a server error occurred                  |

### Get logs for a task

#### Request

GET `/v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="foo"&container="bar"....`

Get the logs for a container running in a task belonging to a service, running in a cluster belonging to an account.  The request can be for just the task and container, in which case up to 10,000 (or 1MB) of the most recent log messages will be returned.  A limit can be passed to limit the number of records returned, and a sequence token, `seq` can be passed to support paging.  Additionally, `start` and `end` times can be passed in milliseconds from the unix epoch.  More details can be found [in the documentation][https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_GetLogEvents.html]

##### Examples

```text
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="foo"&container="bar"
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="foo"&container="bar"&limit="30"
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="foo"&container="bar"&limit="30"&seq="f/35313851203912372440587619261645128276299525300062978048"
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="foo"&container="bar"&start="1583504305223"&end="1583527860973"
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="foo"&container="bar"&start="1583504305223"&end="1583527860973"&limit="30"&seq="f/35313851203912372440587619261645128276299525300062978048"
```

## Managed Task Definitions

### Create a managed task definition

The ECS-API allows for creating a task definition that fits into the service paradigm for Spinup.  This will create a runnable task definition as well as any dependent services such as clusters, secrets and tags.

#### Request

POST /v1/ecs/{account}/taskdefs

```json
{
    "cluster": {
        "clustername": "myclu"
    },
    "taskdefinition": {
        "family": "supercool",
        "cpu": "256",
        "memory": "512",
        "containerdefinitions": [
            {
                "name": "nginx",
                "image": "nginx:alpine",
                "portmappings": [{"ContainerPort": 80}]
            },
            {
                "name": "api",
                "image": "yaleits/testapi",
                "portmappings": [{"ContainerPort": 8080}]
            }
        ]
        },
    "credentials": {
        "api": {
            "SecretString": "{\"username\" : \"myorguser\",\"password\" : \"new-super-sekret-password-string\"}",
        }
    },
    "tags": [
        {
            "Key": "Application",
            "Value": "myclu"
        },
        {
            "Key": "COA",
            "Value": "Bill.Me.Please.COA"
        }
    ]
}
```

#### Response

```json
TODO
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **200 OK**                    | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster or taskdef wasn't found |
| **500 Internal Server Error** | a server error occurred                  |

### Delete a managed task definition

#### Request

DELETE /v1/ecs/{account}/cluster/{cluster}/taskdefs/{taskdef}

#### Response

The response is the service body.

```json
{
    "Cluster": "myclu",
    "TaskDefinition": "supercool-service"
}
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **202 Accepted**              | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster or taskdef wasn't found |
| **500 Internal Server Error** | a server error occurred                  |

### List managed task definitions in a cluster

List returns an array of task definition families - there may be multiple revisions of a task definition.

#### Request

GET /v1/ecs/{account}/cluster/{cluster}/taskdefs

#### Response

The response is the service body.

```json
[
    "supercool"
]
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **200 OK**                    | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster or taskdef wasn't found |
| **500 Internal Server Error** | a server error occurred                  |

### Get a managed task definitions in a cluster

Gets the details about a managed task definition

#### Request

GET /v1/ecs/{account}/cluster/{cluster}/taskdefs/{taskdef}

#### Response

The response is the taskdef body.

```json
TODO
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **200 OK**                    | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster or taskdef wasn't found |
| **500 Internal Server Error** | a server error occurred                  |


### Run a managed task definition in a cluster

Runs a task definition

#### Request

POST /v1/ecs/{account}/cluster/{cluster}/taskdefs/{taskdef}/tasks

The input for running a task uses the RunTaskInput and overrides with our standard required values.  Tags are propagated from the Task Definition.

[RunTaskInput](https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#RunTaskInput)

The `startedBy` parameter is *optional* and can be used to group tasks started by the same process.

```json
{
    "Count": 1,
    "StartedBy": "camden"
}
```

#### Response

The response is the tasks output and any failures.

```json
TODO
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **200 OK**                    | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster or taskdef wasn't found |
| **500 Internal Server Error** | a server error occurred                  |

### Get a list of task definition tasks

#### Request

GET  /v1/ecs/{account}/cluster/{cluster}/taskdefs/{taskdef}/tasks[?status=RUNNING][&status=STOPPED][&startedBy=foobar]

#### Response

The response is the tasks list

```json
[
    "myclu/55a94cb97c234fe8a5af3b64cb14d3ff",
    "myclu/7ee0e0566a234a4eaa2baca61e73c9d6",
    "myclu/acb41329b6cf4a3db06d23d477b386ee"
]
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **200 OK**                    | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster wasn't found            |
| **500 Internal Server Error** | a server error occurred                  |

### Get a details of task definition task

#### Request

GET  /v1/ecs/{account}/cluster/{cluster}/taskdefs/{taskdef}/tasks/{task}

#### Response

The response is the tasks list.

```json
{
    "Tasks": [
        {
            "Attachments": [
                {
                    "Details": [
                        {
                            "Name": "subnetId",
                            "Value": "subnet-aaaaaaa"
                        },
                        {
                            "Name": "networkInterfaceId",
                            "Value": "eni-vvvvvvvvvvvv"
                        },
                        {
                            "Name": "macAddress",
                            "Value": "12:89:50:8d:21:a7"
                        },
                        {
                            "Name": "privateDnsName",
                            "Value": "ip-10-1-2-3.ec2.internal"
                        },
                        {
                            "Name": "privateIPv4Address",
                            "Value": "10.1.2.3"
                        }
                    ],
                    "Id": "69403df1-ca83-480d-8fb2-80891ab065b8",
                    "Status": "DELETED",
                    "Type": "ElasticNetworkInterface"
                }
            ],
            "Attributes": null,
            "AvailabilityZone": "us-east-1d",
            "CapacityProviderName": null,
            "ClusterArn": "arn:aws:ecs:us-east-1:001122334455:cluster/myclu",
            "Connectivity": "CONNECTED",
            "ConnectivityAt": "2021-06-02T16:15:08.273Z",
            "ContainerInstanceArn": null,
            "Containers": [
                {
                    "ContainerArn": "arn:aws:ecs:us-east-1:001122334455:container/myclu/25cb8e5435b04a0396c1fea4db587b8b/6fd8e73b-ccea-4a54-b52d-8f64fafb1f92",
                    "Cpu": "0",
                    "ExitCode": 0,
                    "GpuIds": null,
                    "HealthStatus": "UNKNOWN",
                    "Image": "busybox",
                    "ImageDigest": null,
                    "LastStatus": "STOPPED",
                    "Memory": null,
                    "MemoryReservation": null,
                    "Name": "sleeper",
                    "NetworkBindings": [],
                    "NetworkInterfaces": [
                        {
                            "AttachmentId": "69403df1-ca83-480d-8fb2-80891ab065b8",
                            "Ipv6Address": null,
                            "PrivateIpv4Address": "10.1.2.3"
                        }
                    ],
                    "Reason": null,
                    "RuntimeId": "25cb8e5435b04a0396c1fea4db587b8b-1634946175",
                    "TaskArn": "arn:aws:ecs:us-east-1:001122334455:task/myclu/25cb8e5435b04a0396c1fea4db587b8b"
                },
                {
                    "ContainerArn": "arn:aws:ecs:us-east-1:001122334455:container/myclu/25cb8e5435b04a0396c1fea4db587b8b/bd3ec6ff-0cff-4d06-bfa9-843ba9195b85",
                    "Cpu": "0",
                    "ExitCode": 2,
                    "GpuIds": null,
                    "HealthStatus": "UNKNOWN",
                    "Image": "something/testapi",
                    "ImageDigest": null,
                    "LastStatus": "STOPPED",
                    "Memory": null,
                    "MemoryReservation": null,
                    "Name": "api",
                    "NetworkBindings": [],
                    "NetworkInterfaces": [
                        {
                            "AttachmentId": "69403df1-ca83-480d-8fb2-80891ab065b8",
                            "Ipv6Address": null,
                            "PrivateIpv4Address": "10.1.2.3"
                        }
                    ],
                    "Reason": null,
                    "RuntimeId": "25cb8e5435b04a0396c1fea4db587b8b-946514567",
                    "TaskArn": "arn:aws:ecs:us-east-1:001122334455:task/myclu/25cb8e5435b04a0396c1fea4db587b8b"
                }
            ],
            "Cpu": "256",
            "CreatedAt": "2021-06-02T16:15:04.391Z",
            "DesiredStatus": "STOPPED",
            "ExecutionStoppedAt": "2021-06-02T16:16:48.968Z",
            "Group": "family:supercool",
            "HealthStatus": "UNKNOWN",
            "InferenceAccelerators": null,
            "LastStatus": "STOPPED",
            "LaunchType": "FARGATE",
            "Memory": "512",
            "Overrides": {
                "ContainerOverrides": [
                    {
                        "Command": null,
                        "Cpu": null,
                        "Environment": null,
                        "EnvironmentFiles": null,
                        "Memory": null,
                        "MemoryReservation": null,
                        "Name": "sleeper",
                        "ResourceRequirements": null
                    },
                    {
                        "Command": null,
                        "Cpu": null,
                        "Environment": null,
                        "EnvironmentFiles": null,
                        "Memory": null,
                        "MemoryReservation": null,
                        "Name": "api",
                        "ResourceRequirements": null
                    }
                ],
                "Cpu": null,
                "ExecutionRoleArn": null,
                "InferenceAcceleratorOverrides": [],
                "Memory": null,
                "TaskRoleArn": null
            },
            "PlatformVersion": "1.4.0",
            "PullStartedAt": "2021-06-02T16:15:29.45Z",
            "PullStoppedAt": "2021-06-02T16:16:26.565Z",
            "StartedAt": "2021-06-02T16:16:29.066Z",
            "StartedBy": null,
            "StopCode": "EssentialContainerExited",
            "StoppedAt": "2021-06-02T16:17:24.104Z",
            "StoppedReason": "Essential container in task exited",
            "StoppingAt": "2021-06-02T16:17:10.101Z",
            "Tags": [],
            "TaskArn": "arn:aws:ecs:us-east-1:001122334455:task/myclu/25cb8e5435b04a0396c1fea4db587b8b",
            "TaskDefinitionArn": "arn:aws:ecs:us-east-1:001122334455:task-definition/supercool:43",
            "Version": 5,
            "Revision": 43
        }
    ],
    "Failures": []
}
```

| Response Code                 | Definition                               |
| ----------------------------- | -----------------------------------------|
| **200 OK**                    | okay                                     |
| **400 Bad Request**           | badly formed request                     |
| **404 Not Found**             | account, cluster or taskdef wasn't found |
| **500 Internal Server Error** | a server error occurred                  |


### Stop a task definition task

#### Request

DELETE  /v1/ecs/{account}/cluster/{cluster}/taskdefs/{taskdef}/tasks/{task}

#### Response

The response is the tasks list.

```json
"OK"
```

| Response Code                 | Definition                                 |
| ----------------------------- | -------------------------------------------|
| **200 OK**                    | okay                                       |
| **400 Bad Request**           | badly formed request                       |
| **404 Not Found**             | account, cluster, def or task wasn't found |
| **500 Internal Server Error** | a server error occurred                    |

## SSM Parameters

### Create a param

Parameters are automatically creating in the `org` path.  A `prefix` should be specified in the `Name`.

POST `/v1/ecs/{account}/params/{prefix}`

#### Request

[PutParameterInput](https://docs.aws.amazon.com/sdk-for-go/api/service/ssm/#PutParameterInput)

```json
{
    "Description": "a secret parameter",
    "Name": "newsecret123",
    "Value": "abc123",
    "Tags": [
        {"Key": "MyKey", "Value": "MyValue"},
        {"Key": "Application", "Value": "someprefix"},
    ]
}
```

#### Response

```json
{
    "Tier": "Standard",
    "VersionId": "592CEFAE-7B74-4A22-B1C9-55F958531579"
}
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | account wasn't found                  |
| **500 Internal Server Error** | a server error occurred               |

### List parameters

Listing parameters is limited to the parameters that belong to the *org*. A `prefix` is also required.

GET `/v1/ecs/{account}/params/{prefix}`

#### Response

```json
[
    "newsecret123",
    "newsecret321",
    "oldsecret123"
]
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | account or prefix wasn't found        |
| **500 Internal Server Error** | a server error occurred               |

### Show a parameter

Pass the parameter `prefix` and `param` to get the metadata about a secret.  The `org` will automatically be prepended.

GET `/v1/ecs/{account}/params/{prefix}/{param}`

#### Response

```json
{
    "ARN": "arn:aws:ssm:us-east-1:001122334455:parameter/myorg/someprefix/newsecret123",
    "Name": "newsecret123",
    "Description": "a test secret shhhhhh! 123",
    "KeyId": "arn:aws:kms:us-east-1:001122334455:key/aaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee",
    "Type": "SecureString",
    "Tags": [
        {
            "Key": "MyKey",
            "Value": "MyValue"
        },
        {
            "Key": "spinup:org",
            "Value": "myorg"
        },
        {
            "Key": "Application",
            "Value": "someprefix"
        }
    ],
    "LastModifiedDate": "2019-10-09 15:43:44 +0000 UTC",
    "Version": 1
}
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | account, param or prefix wasn't found |
| **500 Internal Server Error** | a server error occurred               |

### Delete a parameter

DELETE `/v1/ecs/{account}/params/{prefix}/{param}`

#### Response

```json
{"OK"}
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | account, param or prefix wasn't found |
| **500 Internal Server Error** | a server error occurred               |

### Delete all parameters in a prefix

DELETE `/v1/ecs/{account}/params/{prefix}`

#### Response

```json
{
    "Message": "OK",
    "Deleted": 3,
}
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | account or prefix wasn't found        |
| **500 Internal Server Error** | a server error occurred               |

### Update a parameter

Update the tags and/or value of a parameter.  Pass the `prefix` and the `param`.

PUT `/v1/ecs/{account}/params/{prefix}/{param}`

#### Request

```json
{
    "Value": "abc123",
    "Tags": [
        {"Key": "MyKey", "Value": "MyValue"},
        {"Key": "Application", "Value": "someprefix"},
    ]
}
```

#### Response

```json
{"OK"}
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | account, param or prefix wasn't found |
| **500 Internal Server Error** | a server error occurred               |

## Secrets

`Secrets` store binary or string data in AWS secrets manager. By default, secrets are encrypted (in AWS) by the `defaultKmsKeyId` given for each `account`.

### Create a secret

POST `/v1/ecs/{account}/secrets`

#### Request

```json
{
    "Name": "sshhhhh",
    "SecretString": "abc123"
}
```

#### Response

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:001122334455:secret:sshhhhh-Z8CxfW",
    "Name": "sshhhhh",
    "VersionId": "592CEFAE-7B74-4A22-B1C9-55F958531579"
}
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | okay                            |
| **400 Bad Request**           | badly formed request            |
| **500 Internal Server Error** | a server error occurred         |

### List secrets

Listing secrets is limited to the secrets that belong to the *org*. Optionally pass `key=value` pairs
to filter on secret tags.

GET `/v1/ecs/{account}/secrets[?key1=value1[&key2=value2&key3=value3]]`

#### Response

```json
[
    "arn:aws:secretsmanager:us-east-1:001122334455:secret:TopSekritPassword-rJ93nm",
    "arn:aws:secretsmanager:us-east-1:001122334455:secret:ShhhDontTellAnyone-123-BFyDco"
]
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | okay                            |
| **400 Bad Request**           | badly formed request            |
| **500 Internal Server Error** | a server error occurred         |

### Show a secret

Pass the secret id to get the metadata about a secret.

GET `/v1/ecs/{account}/secret/{secret}`

#### Response

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:001122334455:secret:ShhhDontTellAnyone-123-BFyDco",
    "DeletedDate": null,
    "Description": null,
    "KmsKeyId": null,
    "LastAccessedDate": null,
    "LastChangedDate": "2019-07-01T21:30:54Z",
    "LastRotatedDate": null,
    "Name": "ShhhDontTellAnyone",
    "RotationEnabled": null,
    "RotationLambdaARN": null,
    "RotationRules": null,
    "Tags": [
        {
            "Key": "spinup:org",
            "Value": "localdev"
        }
    ],
    "VersionIdsToStages": {
        "12345678-9012-3456-7898-123456789012": [
            "AWSCURRENT"
        ]
    }
}
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | okay                            |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | secret wasn't found in the org  |
| **500 Internal Server Error** | a server error occurred         |

### Delete a secret

Pass the secret id and an options `window` parameter (in days).  A parameter of `0` will cause the secret
to be deleted immediately.  Otherwise the grace period must be between `7` and `30`.

DELETE `/v1/ecs/{account}/secret/{secret}[?window=[0|7-30]]`

#### Response

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:001122334455:secret:ShhhDontTellAnyone-123-BFyDco",
    "DeletionDate": "2019-07-13T11:18:33Z",
    "Name": "ShhhDontTellAnyone"
}
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | okay                            |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | secret wasn't found in the org  |
| **500 Internal Server Error** | a server error occurred         |

### Update a secret

Pass the secret id, the new secret string value and/or the list of tags to update. Currently,
updating binary secrets is not supported, nor is setting the secret version.

PUT `/v1/ecs/{account}/secrets/{secret}`

#### Request

```json
{
    "Secret": "TOPSEKRETsshhhhh",
    "Tags": [
        {
            "key": "Application",
            "value": "FooBAAAAAR"
        }
    ]
}
```

#### Response

When only updating tags, you will get an empty response on success. When updating a secret:

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:001122334455:secret:ShhhDontTellAnyone-123-BFyDco",
    "Name": "ShhhDontTellAnyone",
    "VersionId": "AWSCURRENT"
}
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | okay                            |
| **400 Bad Request**           | badly formed request            |
| **404 Not Found**             | secret wasn't found in the org  |
| **500 Internal Server Error** | a server error occurred         |

### List load balancer target groups for a space

GET `/v1/ecs/{account}/lbs?space={space}`

#### Response

```json
{
    "test-tg-1": "arn:aws:elasticloadbalancing:us-east-1:001122334455:targetgroup/test-tg-1/0987654321",
    "test-tg-2": "arn:aws:elasticloadbalancing:us-east-1:001122334455:targetgroup/test-tg-2/0987654321"
}
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **500 Internal Server Error** | a server error occurred               |

### Describe load balancers in a space

Returns a list of load balancers in a given space. For each load balancer we also get a list of configured listeners and their rules, as well as any associated target groups and the health status of their targets.

GET `/v1/ecs/{account}/lbs/{space}`

#### Response

```json
[
    {
        "LoadBalancerArn": "arn:aws:elasticloadbalancing:us-east-1:012345678901:loadbalancer/app/zee-special-alb/1b12a9c2316583d9",
        "LoadBalancerName": "zee-special-alb",
        "LoadBalancerType": "application",
        "DNSName": "internal-zee-special-alb-123456789.us-east-1.elb.amazonaws.com",
        "Listeners": [
            {
                "ListenerArn": "arn:aws:elasticloadbalancing:us-east-1:012345678901:listener/app/zee-special-alb/1b12a9c2316583d9/64e675c6f7554ed1",
                "ListenerName": "HTTP:80",
                "Rules": [
                    {
                        "RuleArn": "arn:aws:elasticloadbalancing:us-east-1:012345678901:listener-rule/app/zee-special-alb/1b12a9c2316583d9/64e675c6f7554ed1/8980e764316b7e1b",
                        "If": "Default",
                        "Then": "redirect to HTTPS://#{host}:443/#{path}?#{query}"
                    }
                ]
            },
            {
                "ListenerArn": "arn:aws:elasticloadbalancing:us-east-1:012345678901:listener/app/zee-special-alb/1b12a9c2316583d9/998b8bb49534c4c7",
                "ListenerName": "HTTPS:443",
                "SslPolicy": "ELBSecurityPolicy-TLS-1-2-2017-01",
                "Rules": [
                    {
                        "RuleArn": "arn:aws:elasticloadbalancing:us-east-1:012345678901:listener-rule/app/zee-special-alb/1b12a9c2316583d9/998b8bb49534c4c7/1c75ae42a6a1e567",
                        "If": "host-header: special.internal.example.com",
                        "Then": "forward to tf-20211109200946929800000001",
                        "TargetGroups": [
                            {
                                "TargetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:012345678901:targetgroup/tf-20211109200946929800000001/60cd460879687ff5",
                                "TargetGroupName": "tf-20211109200946929800000001",
                                "TargetType": "ip",
                                "Targets": [
                                    {
                                        "Id": "10.20.30.40",
                                        "Port": "443",
                                        "State": "healthy",
                                        "Reason": ""
                                    }
                                ]
                            }
                        ]
                    },
                    {
                        "RuleArn": "arn:aws:elasticloadbalancing:us-east-1:012345678901:listener-rule/app/zee-special-alb/1b12a9c2316583d9/998b8bb49534c4c7/89e4837fe23d18c3",
                        "If": "Default",
                        "Then": "fixed response: status code 412"
                    }
                ]
            }
        ]
    }
]
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **500 Internal Server Error** | a server error occurred               |

## Development

- Install Go v1.11 or newer
- Enable Go modules: `export GO111MODULE=on`
- Create a config: `cp -p config/config.example.json config/config.json`
- Edit `config.json` and update the parameters
- Run `go run .` to start the app locally while developing
- Run `go test ./...` to run all tests
- Run `go build ./...` to build the binary

## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

GNU Affero General Public License v3.0 (GNU AGPLv3)
Copyright (c) 2019-2022 Yale University
