[![Go Report Card](https://goreportcard.com/badge/github.com/verrazzano/verrazzano)](https://goreportcard.com/report/github.com/verrazzano/verrazzano)

# Verrazzano Platform Operator

Instructions for building and testing the Verrazzano platform operator.

## Prerequisites
* `kustomize` v3.1.0+
* `kubebuilder` 2.3.1+
* `go` version v1.15.7
* Docker
* `kubectl`

## Build and install Verrazzano

To build and then install Verrazzano using the Verrazzano platform operator:

```
# Create the verrazzano-platform-operator image.  For now, the image needs public access.
# Replace <docker-repository> and <namespace for image>.
$ export DOCKER_REPO=<docker-repository>
$ export DOCKER_NAMESPACE=<namespace for image>
$ make docker-push

# Create the verrrazzano-platform-operator deployment YAML file.
# Define the VZ_DEV_IMAGE env variable and call the create-test-deploy target
# - Replace <verrazzano-image> with the verrazzano-platform-operator image you created with `make docker-push`
# - Creates a valid deployment YAML file in build/deploy/operator.yaml
$ export VZ_DEV_IMAGE=<verrazzano-image>
$ make create-test-deploy

# Deploy the verrazzano-platform-operator
$ kubectl apply -f build/deploy/operator.yaml

# Verify verrazzano-platform-operator pod is running
$ kubectl get pods -n verrazzano-install

# Initiate a Verrazzano install for nip.io
$ kubectl apply -f config/samples/install-default.yaml

# NOTE:  If you chose to deploy a cluster that makes use of OCI DNS perform the following instead of the nip.io
# cluster deployment command:

# Generate a secret named "oci" based on the OCI configuration profile you wish to leverage.  You
# can specify a profile other than DEFAULT and a different secret name if you wish.  See instruction by running
# ./scripts/install/create_oci_config_secret.sh
$ ./scripts/install/create_oci_config_secret.sh

# Copy the config/samples/install-oci.yaml file
$ cp config/samples/install-oci.yaml /tmp

# Edit the file and provide the DNS ZONE name, OCID, and compartment OCID, and secret name

# Monitor the install
$ kubectl logs -f $(kubectl get pod -l job-name=verrazzano-install-my-verrazzano -o jsonpath="{.items[0].metadata.name}")

# Wait for the Verrazzano install to complete
$ kubectl wait --timeout=20m --for=condition=InstallComplete verrazzano/my-verrazzano
```

To uninstall Verrazzano using the Verrazzano platform operator:

```
# Initiate a Verrazzano uninstall
$ kubectl delete -f config/samples/install-default.yaml

# Monitor the uninstall
$ kubectl logs -f $(kubectl get pod -l job-name=verrazzano-uninstall-my-verrazzano -o jsonpath="{.items[0].metadata.name}")
```

## Build the Verrazzano platform operator

- To generate manifests (for example, CRD, RBAC, and such):

    ```
    $ make manifests
    ```

- To generate code (for example, `zz_generated.deepcopy.go`):

    ```
    $ make generate
    ```

- To do all the source code checks, such as `fmt`, `lint`, and such:

    ```
    $ make check
    ```

- To build the operator and generated code:

    ```
    $ make go-install
    ```

## Test the Verrazzano platform operator

You need a Kubernetes cluster to run against.

* Install the CRDs into the cluster:
    ```
    $ make install-crds
    ```

* Run the operator. This will run in the foreground, so switch to a new terminal if you want to leave it running.
    ```
    $ make run
    ```

* Create a custom resource.  You will notice that messages are logged to the operator
when the custom resource is applied.
    ```
    $ kubectl apply -f config/samples/install-default.yaml
    ```

* Delete the custom resource.  You will notice that messages are logged to the operator
when the custom resource is deleted.
    ```
    $ kubectl delete -f config/samples/install-default.yaml
    ```
* Uninstall the CRDs from the cluster:
    ```
    $ make uninstall-crds
    ```

## Build and push Docker images

* To build the Docker image:
    ```
    $ make docker-build
    ```
* To build and push the Docker image:
    ```
    $ make docker-push
    ```
