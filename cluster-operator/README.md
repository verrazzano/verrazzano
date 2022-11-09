[![Go Report Card](https://goreportcard.com/badge/github.com/verrazzano/verrazzano)](https://goreportcard.com/report/github.com/verrazzano/verrazzano)

# Verrazzano Cluster Operator

## Prerequisites
* `kustomize` v3.1.0+
* `kubebuilder` 2.3.1+
* `go` version v1.15.7
* Docker
* `kubectl`

## Build the Verrazzano cluster operator
* To generate DeepCopy and DeepCopyInto methods for changes in types in APIs
    ```
    make generate
    ```

* To generate manifests, for example, CRD, RBAC, and such:
    ```
    make manifests
    ```

* To do all the source code checks, such as `fmt`, `lint`, and such:
    ```
    make check
    ```

* To build the operator and generated code:
    ```
    make go-install
    ```

## Test the Verrazzano cluster operator

You need a Kubernetes cluster to run against.

* Install the CRDs into the cluster:
    ```
    make install-crds
    ```

* Run the operator. This will run in the foreground, so switch to a new terminal if you want to leave it running.
    ```
    make run
    ```

* Uninstall the CRDs from the cluster:
    ```
    make uninstall-crds
    ```

## Build and push Docker images

* To build the Docker image:
    ```
    make docker-build

* To push the Docker image:
    ```
    make docker-push
    ```  
