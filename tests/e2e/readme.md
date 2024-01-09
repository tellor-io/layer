<!-- TODO: Edit -->
# Instructions

## Required Software

[Docker](https://docs.docker.com/get-docker/)
[Minikube](https://minikube.sigs.k8s.io/docs/start/)
[kubectl](https://kubernetes.io/releases/download/)

- docker is a platform to develop and run applications in isolation.

- minikube is a tool for running Kubernetes locally. It sets up a single-node Kubernetes cluster on your machine.

```sh
minikube start
```

```sh
minikube stop
```

- To reset minikube:

```sh
minikube delete
```

kubectl is a command-line tool for controlling Kubernetes clusters.

- Some helpful commands:

```sh
kubectl get pods [--namespace test]
```

```sh
# Prints logs from a specified container in the pod.
kubectl logs <pod-id> [--container=container-name]
```

```sh
# Provides detailed information about a specific pod.
kubectl describe pod <pod-name> [--namespace <namespace-name>]
```

```sh
# See what namespaces exist
kubectl get namespaces
```

```sh
# Create namespace
kubectl create namespace test
```

### Some definitions/explanations

- kubernetes: software that helps manage and orchestrate docker containers.
- namespace: creating a namespace allows for dividing a cluster into smaller isolated groupings that have their own resources/policies.
- pods: A pod is the basic unit in kubernetes that one or more containers operate in, similar to a VM.

#### Running the simple test

```sh
# note: test takes some time to run, to skip delete E2E=true from cmd
E2E=true KNUU_NAMESPACE=test go test ./tests/e2e/... -timeout 30m -v 
```
