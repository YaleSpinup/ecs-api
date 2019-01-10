# ecs-api

This API provides simple restful API access to Amazon's ECS Fargate service.

## Endpoints

```
GET /v1/ecs/ping
GET /v1/ecs/version

// Service Orchestration handlers
POST /v1/ecs/{account}/services
DELETE /v1/ecs/{account}/services

// Clusters handlers
GET /v1/ecs/{account}/clusters
POST /v1/ecs/{account}/clusters
GET /v1/ecs/{account}/clusters/{cluster}
DELETE /v1/ecs/{account}/clusters/{cluster}

// Services handlers
GET /v1/ecs/{account}/clusters/{cluster}/services[?all=true]
POST /v1/ecs/{account}/clusters/{cluster}/services
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}
DELETE /v1/ecs/{account}/clusters/{cluster}/services/{service}
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="foo"&container="bar"

// Tasks handlers
GET /v1/ecs/{account}/clusters/{cluster}/tasks
POST /v1/ecs/{account}/clusters/{cluster}/tasks
GET /v1/ecs/{account}/clusters/{cluster}/tasks/{task}
DELETE /v1/ecs/{account}/clusters/{cluster}/tasks/{task}

// Task definitions handlers
GET /v1/ecs/{account}/taskdefs
POST /v1/ecs/{account}/taskdefs
GET /v1/ecs/{account}/taskdefs/{taskdef}
DELETE /v1/ecs/{account}/taskdefs/{taskdef}

GET /v1/ecs/{account}/servicediscovery/services
POST /v1/ecs/{account}/servicediscovery/services
GET /v1/ecs/{account}/servicediscovery/services/{id}
DELETE /v1/ecs/{account}/servicediscovery/services/{id}
```

## Orchestration

The service orchestration endpoints for creating and deleting services allow building and destroying services with one call to the API.

The endpoints are essentially wrapped versions of the ECS and ServiceDiscovery endpoints from AWS.  The endpoint will determine
what has been provided and try to take the most logical action.  For example, if you provide `CreateClusterInput`, `RegisterTaskDefinitionInput`
and `CreateServiceInput`, the API will attempt to create the cluster, then the task definition and then the service using the created
resources.  If you only provide the `CreateServiceInput` with the task definition name, the cluster name and the service registries, it
will assume those resources already exist and just try to create the service.

Example request body of new cluster, new task definition, new service registry and new service:

```json
{
    "cluster": {
        "clustername": "myclu"
    },
    "taskdefinition": {
        "family": "mytaskdef",
        "cpu": "256",
        "memory": "512",
        "containerdefinitions": [
            {
                "name": "webserver",
                "image": "nginx:alpine",
                "ports": [80,443]
            }
        ]
    },
    "service": {
        "desiredcount": 1,
        "servicename": "webapp"
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
    }
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
                "registryarn": "arn:aws:servicediscovery:us-east-1:12345678910:service/srv-tvtbgvkkxtts3qlf"
            }
        ]
    }
}
```

## Clusters

Clusters provide groupings of containers.  With `FARGATE`, clusters are simply created with a name parameter.

To create a `cluster`, just `POST` the name to the endpoint:

```json
{
    "name": "myCluster"
}
```

## Services

`Services` are long lived/scalable tasks launched based on a task definition.  Services can be comprised of multiple `containers` (because multiple containers can be defined in a task definition!).  AWS will restart containers belonging to services when they exit or die.  `Services` can be scaled and can be associated with load balancers.  Tasks that make up a service can have IP addresses, but those addresses change when tasks are restarted.

To create a `service`, just `POST` to the endpoint:

```json
{}
```

## Tasks

`Tasks`, in contrast to `Services`, should be considered short lived.  They will not automatically be restarted or scaled by AWS and cannot be associated with a load balancer.  `Tasks`, as with `services`, can be made up of multiple containers.  Tasks can have IP addresses, but those addresses change when tasks are restarted.

To create a `task`, just `POST` to the endpoint:

```json
{}
```

## Task Definitions

`Task definitions` describe a set of containers used in a `service` or `task`.

To create a `task definition`, just `POST` to the endpoint:

```json
{}
```

## Author

E Camden Fisher <camden.fisher@yale.edu>

## License

The MIT License (MIT)

Copyright (c) 2018 Yale University

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
