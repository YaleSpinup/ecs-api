# ecs-api

This API provides simple restful API access to Amazon's ECS Fargate service.

## Endpoints

```
GET /v1/ecs/ping
GET /v1/ecs/version

// Clusters handlers
GET /v1/ecs/{account}/clusters
POST /v1/ecs/{account}/clusters
GET /v1/ecs/{account}/clusters/{cluster}
DELETE /v1/ecs/{account}/clusters/{cluster}

// Services handlers
GET /v1/ecs/{account}/clusters/{cluster}/services
POST /v1/ecs/{account}/clusters/{cluster}/services
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}
DELETE /v1/ecs/{account}/clusters/{cluster}/services/{service}

// Tasks handlers
GET /v1/ecs/{account}/clusters/{cluster}/tasks
POST /v1/ecs/{account}/clusters/{cluster}/tasks
GET /v1/ecs/{account}/clusters/{cluster}/tasks/{task}
DELETE /v1/ecs/{account}/clusters/{cluster}/tasks/{task}

// Task definitions handlers
GET /v1/ecs{account}//taskdefs
POST /v1/ecs/{account}/taskdefs
GET /v1/ecs/{account}/taskdefs/{taskdef}
DELETE /v1/ecs/{account}/taskdefs/{taskdef}
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