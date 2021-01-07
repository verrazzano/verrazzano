[![Go Report Card](https://goreportcard.com/badge/github.com/verrazzano/verrazzano-application-operator)](https://goreportcard.com/report/github.com/verrazzano/verrazzano-application-operator)

# verrazzano-application-operator

## Prerequisites
* kustomize v3.1.0+
* kubebuilder 2.3.1+
* go version v1.13+
* docker
* kubectl

## Building the Operator

* To generate manifests e.g. CRD, RBAC etc.
    ```
    make manifests
    ```

* To do all the source code checks, such as fmt, lint, etc
    ```
    make check
    ```

* To build the operator and generated code
    ```
    make go-install
    ```

## Testing out the Operator

You’ll need a Kubernetes cluster to run against.

* Install the CRDs into the cluster:
    ```
    make install-crds
    ```

* Run the operator (this will run in the foreground, so switch to a new terminal if you want to leave it running):
    ```
    make run
    ```
  
* Run the operator with webhooks:
  * Run installer/scripts/3-install-vz-oam.sh to create build/webhook-certs 
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

## Building and pushing docker images

* To build the docker image:
    ```
    make docker-build

* To push the docker image:
    ```
    make docker-push
    ```  

## Running kind based integration tests

  make build  
  make docker-build  
  make integ-test  

## Installing the OAM runtime and the Verrazzano application operator

> **NOTE**: These are temporary install/uninstall scripts that will be removed before this repo is made public.

The `installer` directory has scripts that will
install/uninstall both the OAM runtime and the Verrazzano application operator along
with the custom Verrazzano application operator CRDs (e.g. traits).

First create the github packages secret in the verrazzano-system namespace:

```
kubectl create ns verrazzano-system
kubectl create secret -n verrazzano-system  docker-registry github-packages --docker-username=<user@foo.com> --docker-password=<xyz> --docker-server=ghcr.io
```

To install, set the env var for the application operator image, then run the install script.  For example:
```
export VERRAZZANO_APP_OP_IMAGE=ghcr.io/verrazzano/verrazzano-application-operator-jenkins:0.5.0-20201030125421-f7e021b
./installer/install.sh
```

To uninstall, run the following:
```
./installer/uninstall.sh
```
