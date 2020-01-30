# ecs-api

This API provides simple restful API access to Amazon's ECS Fargate service.

## Endpoints

```
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
GET /v1/ecs/{account}/clusters/{cluster}/services/{service}/logs?task="foo"&container="bar"

// Tasks handlers
GET /v1/ecs/{account}/clusters/{cluster}/tasks/{task}

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
```

## Docker Image verification

Image verification checks if an image is available.  Currently, this endpoint is a simple `HEAD` request.  Successful http response
codes mean the image is available.

### Check if an image is available

HEAD `/v1/ecs/images?image={image}`

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
        "clustername": "myclu",
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
            "Description": "myapp-webserver-cred",
            "tags": [
                {"Key": "Application", "Value": "myapp" }
            ]
        }
    },
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

### Orchestrate a service update

Service update orchestration currently supports updating tags and/or forcing a redeployment without changing the service.

PUT `/v1/ecs/{account}/clusters/{cluster}/services/{service}`

##### Request

```json
{
    "ForceNewDeployment": true,
    "Tags": [
        {"Key": "MyKey", "Value": "MyValue"},
        {"Key": "Application", "Value": "someprefix"},
    ]
}
```

##### Response

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

DELETE `/v1/ecs/{account}/clusters/{cluster}/services/{service}[?recursive=true]`

##### Response

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

## Adhoc Requests

### Clusters

Clusters provide groupings of containers.  With `FARGATE`, clusters are simply created with a name parameter.

### Services

`Services` are long lived/scalable tasks launched based on a task definition.  Services can be comprised of multiple `containers` (because multiple containers can be defined in a task definition!).  AWS will restart containers belonging to services when they exit or die.  `Services` can be scaled and can be associated with load balancers.  Tasks that make up a service can have IP addresses, but those addresses change when tasks are restarted.

### Tasks

`Tasks`, in contrast to `Services`, should be considered short lived.  They will not automatically be restarted or scaled by AWS and cannot be associated with a load balancer.  `Tasks`, as with `services`, can be made up of multiple containers.  Tasks can have IP addresses, but those addresses change when tasks are restarted.

### Task Definitions

`Task definitions` describe a set of containers used in a `service` or `task`.

### SSM Parameters

`Parameters` store string data in AWS SSM parameter store. By default, parameters are encrypted (in AWS) by the `defaultKmsKeyId` given for each `account`.

#### Create a param

Parameters are automatically creating in the `org` path.  A `prefix` should be specified in the `Name`.

POST `/v1/ecs/{account}/params/{prefix}`

##### Request

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

##### Response

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

#### List parameters

Listing parameters is limited to the parameters that belong to the *org*. A `prefix` is also required.

GET `/v1/ecs/{account}/params/{prefix}`

##### Response

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

#### Show a parameter

Pass the parameter `prefix` and `param` to get the metadata about a secret.  The `org` will automatically be prepended.

GET `/v1/ecs/{account}/params/{prefix}/{param}`

##### Response

```json
{
    "ARN": "arn:aws:ssm:us-east-1:1234567890:parameter/myorg/someprefix/newsecret123",
    "Name": "newsecret123",
    "Description": "a test secret shhhhhh! 123",
    "KeyId": "arn:aws:kms:us-east-1:1234567890:key/aaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee",
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

#### Delete a parameter

DELETE `/v1/ecs/{account}/params/{prefix}/{param}`

##### Response

```json
{"OK"}
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | account, param or prefix wasn't found |
| **500 Internal Server Error** | a server error occurred               |

#### Delete all parameters in a prefix

DELETE `/v1/ecs/{account}/params/{prefix}`

##### Response

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

#### Update a parameter

Update the tags and/or value of a parameter.  Pass the `prefix` and the `param`.

PUT `/v1/ecs/{account}/params/{prefix}/{param}`

##### Request

```json
{
    "Value": "abc123",
    "Tags": [
        {"Key": "MyKey", "Value": "MyValue"},
        {"Key": "Application", "Value": "someprefix"},
    ]
}
```

##### Response

```json
{"OK"}
```

| Response Code                 | Definition                            |
| ----------------------------- | --------------------------------------|
| **200 OK**                    | okay                                  |
| **400 Bad Request**           | badly formed request                  |
| **404 Not Found**             | account, param or prefix wasn't found |
| **500 Internal Server Error** | a server error occurred               |

### Secrets

`Secrets` store binary or string data in AWS secrets manager. By default, secrets are encrypted (in AWS) by the `defaultKmsKeyId` given for each `account`.

#### Create a secret

POST `/v1/ecs/{account}/secrets`

##### Request

```json
{
    "Name": "sshhhhh",
    "SecretString": "abc123"
}
```

##### Response

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:012345678901:secret:sshhhhh-Z8CxfW",
    "Name": "sshhhhh",
    "VersionId": "592CEFAE-7B74-4A22-B1C9-55F958531579"
}
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | okay                            |
| **400 Bad Request**           | badly formed request            |
| **500 Internal Server Error** | a server error occurred         |

#### List secrets

Listing secrets is limited to the secrets that belong to the *org*. Optionally pass `key=value` pairs
to filter on secret tags.

GET `/v1/ecs/{account}/secrets[?key1=value1[&key2=value2&key3=value3]]`

##### Response

```json
[
    "arn:aws:secretsmanager:us-east-1:012345678901:secret:TopSekritPassword-rJ93nm",
    "arn:aws:secretsmanager:us-east-1:012345678901:secret:ShhhDontTellAnyone-123-BFyDco"
]
```

| Response Code                 | Definition                      |
| ----------------------------- | --------------------------------|
| **200 OK**                    | okay                            |
| **400 Bad Request**           | badly formed request            |
| **500 Internal Server Error** | a server error occurred         |

#### Show a secret

Pass the secret id to get the metadata about a secret.

GET `/v1/ecs/{account}/secret/{secret}`

##### Response

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:012345678901:secret:ShhhDontTellAnyone-123-BFyDco",
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

#### Delete a secret

Pass the secret id and an options `window` parameter (in days).  A parameter of `0` will cause the secret
to be deleted immediately.  Otherwise the grace period must be between `7` and `30`.

DELETE `/v1/ecs/{account}/secret/{secret}[?window=[0|7-30]]`

##### Response

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:012345678901:secret:ShhhDontTellAnyone-123-BFyDco",
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

#### Update a secret

Pass the secret id, the new secret string value and/or the list of tags to update. Currently,
updating binary secrets is not supported, nor is setting the secret version.

PUT `/v1/ecs/{account}/secrets/{secret}`

##### Request

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

##### Response

When only updating tags, you will get an empty response on success. When updating a secret:

```json
{
    "ARN": "arn:aws:secretsmanager:us-east-1:012345678901:secret:ShhhDontTellAnyone-123-BFyDco",
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
Copyright (c) 2019 Yale University
