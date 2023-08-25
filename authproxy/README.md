[![Go Report Card](https://goreportcard.com/badge/github.com/verrazzano/verrazzano)](https://goreportcard.com/report/github.com/verrazzano/verrazzano)

# Verrazzano Authproxy

## Prerequisites
* `go` version v1.19.3
* Docker
* `kubectl`

* To build the authproxy:
    ```
    make go-install
    ```

## Test the Verrazzano authproxy

You need a Kubernetes cluster to run against.

* Run the authproxy. This will run in the foreground, so switch to a new terminal if you want to leave it running.
    ```
    make run
    ```

## Build and push Docker images

* To build the Docker image:
    ```
    make docker-build
    ```

* To push the Docker image:
    ```
    make docker-push
    ```  
