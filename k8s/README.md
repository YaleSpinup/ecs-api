# k8s development Readme

The application ships with a basic k8s config (currently only configured for development) in the `k8s/` directory.  There you will find a `Dockerfile` and yaml configuration to deploy the *ecsapi* pod, service and ingress.  There is also an example configuration yaml (`k8s-config.yaml`) which needs to be populated by you before skaffold can deploy the ecsapi.

## install minikube

[Install minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)

## install skaffold

[Install skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold)

## setup ingress controller (do this once on your cluster)

```
kubectl apply -f https://gist.githubusercontent.com/fishnix/a94dd54ec72523024f5a0b99ae7c6e49/raw/013f86ab7af23eb014f25ba18e5d24c4fd329689/traefik-rbac.yaml
kubectl apply -f https://gist.githubusercontent.com/fishnix/a94dd54ec72523024f5a0b99ae7c6e49/raw/013f86ab7af23eb014f25ba18e5d24c4fd329689/traefik-ds.yaml
```

## create k8s secret config

* modify the local configuration file in `config/config.json`

* copy example secret yaml `cp k8s/example-k8s-config.yaml k8s/k8s-config.yaml`

* base64 encode the configuration `cat config/config.json | base64 -w0`

* copy output of `config.json` secret into `k8s-config.yaml`

## develop

* run `skaffold dev` in the root of the project

* run `minikube ip` to get the ip of your minikube vm

* use the endpoint `http://<<minikube_ip>>/v1/ecs`

Saving your code should rebuild and redeploy your project automatically

## [non-]profit
