[![Go Report Card](https://goreportcard.com/badge/github.com/verrazzano/verrazzano-application-operator)](https://goreportcard.com/report/github.com/verrazzano/verrazzano-application-operator)

# Verrazzano Application Operator

## Prerequisites
* `kustomize` v3.1.0+
* `kubebuilder` 2.3.1+
* `go` version v1.15.7
* Docker
* `kubectl`

## Build the Verrazzano application operator
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

## Test the Verrazzano application operator

You need a Kubernetes cluster to run against.

* Install the CRDs into the cluster:
    ```
    make install-crds
    ```

* Run the operator. This will run in the foreground, so switch to a new terminal if you want to leave it running.
    ```
    make run
    ```

* Run the operator with webhooks:
  * Run `installer/scripts/4-install-vz-oam.sh` to create `build/webhook-certs`
  * Run the operator with webhooks enabled:
    ```
    go run main.go --kubeconfig=${KUBECONFIG} --cert-dir=build/webhook-certs
    ```
  * Test the webhook endpoint:
    ```
    curl -X POST --insecure \
       https://localhost:9443/appconfig-defaulter \
       -H 'Content-Type: application/json' \
       -d @test/integ/testdata/hello-app_appconfig-defaulter-request.json
    ```

* Create a custom resource.  You will notice that messages are logged to the operator
when the custom resource is applied.
    ```
    kubectl apply -f config/samples/
    ```

* Delete the custom resource.  You will notice that messages are logged to the operator
when the custom resource is deleted.
    ```
    kubectl delete -f config/samples/
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
