# k8s development Readme

The application ships with a basic k8s config (currently only configured for development) in the `k8s/` directory.  There you will find a `Dockerfile` and yaml configuration to deploy the *ecsapi* pod, servive and ingress.  There is also an example configuration yaml which needs to be populated by you before skaffold can deploy the ecsapi. 

## install minikube

[Install minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)

## install skaffold

[Install skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold)

## create k8s secret config

* Modify the local configuration file in `config/config.json`

* `cat config/config.json | base64`

* copy output into `config.json` secret in `k8s-config.yaml`

## develop

* run `skaffold dev` in the root of the project

* run `minikube ip` to get the ip of your minikube vm

* use the endpoint `http://<<minikube_ip>>/v1/ecs`

Saving your code should rebuild and deploy your project automatically

## [non-]profit