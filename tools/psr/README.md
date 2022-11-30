# PSR Testing Tool
The PSR testing tool is used to test Verrazzano performance, scalability, and
reliability. The tool works by doing some work on the Verrazzano cluster, then collecting 
and analyzing results. The type of work that is done depends on the goal of the PSR test.  

## Concepts
At a high level, you run a test scenario using the `psrctl` command line interface to do some aspect of
PSR testing, whether it involves stressing a component, doing long term testing for
resource leaks, or other testing.  

### Worker
A worker is the code that performs a specific task, such as putting logs into OpenSearch.  A worker
does a single task and runs continuously in a loop doing the same task repeatedly.  Workers are
deployed in a pod and run in the target cluster that you are testing.  You can have multiple worker 
threads (doing the same work), along with multiple replicas.  The type of work done would determine 
how many workers of the same type should be running.  The idea is that different types of worker are 
combined into scenarios, which is described later.

### Backend
Workers run in backend pods that are deployed using Helm charts. The backend consists of a single image that has 
all the worker code for all workers.  When a pod is started, the worker config is passed in as a set of Env vars.
The main.go code in the pod gets the worker type, creates an instance of the worker, then invokes it to run. 
The pod only runs a single worker, which executes until the pod terminates.  By default, the worker runs forever.

### Use Case
The term `use case` describes what work is being done by a worker. All use cases are deployed using 
the [worker Helm chart](./manifests/charts/worker).  Each worker type has a Helm values override YAML file 
that provides worker specific configuration, such as [usecases/opensearch/getlogs.yaml](./manifests/usecases/opensearch/getlogs.yaml).  
To run a use case, just do a Helm install. For example, to deploy a use case to generate logs using 10 replicas, 
you would run the following command:
```
helm install psr-writelogs manifests/charts/worker -f manifests/usecases/opensearch/writelogs.yaml --set replicas=10
```

### Scenario
A scenario is a collection of use cases with a curated configuration, that are run concurrently.
An example is a scenario where one use case generates logs and another scales OpenSearch out and in, repeatedly.
The PSR command line interface, `psrctl`, is used to run scenarios.  See [psrctl README](./psrctl/README.md)

## Usage
Use the Makefile to build the backend image or execute other targets. If you
just want to try the example use case on a local Kind cluster, then run the following which
builds the code, builds the docker image, loads the docker image into the Kind cluster, and deploys the example use case.
```
make run-example-k8s
```
After you run the make command, run `helm list` to see the `psr` release. Then use kubectl to 
get the backend pod logs in the default namespace to see that they are generating log messages.

**NOTE:** Any Kubernetes platform can be used, such as OKE.

The other Make targets include:  
* go-build - build the code
* go-build-cli - build the psrctl CLI
* docker-clean - remove artifacts from a previous build
* docker-build - calls go-build, then build the image
* docker-push - calls docker-build, then push the image to ghcr.io.
* kind-load-image - calls docker-build, then load the image to a local Kind cluster
* run-example-oam - calls kind-load-image, then deploy the example worker as an OAM app
* run-example-k8s - calls kind-load-image, then deploy the example worker as a Kubernetes deployment
* install-cli - build and install the psrctl CLI to your go path

### Running scenarios
Refer to [psrctl README](./psrctl/README.md) for information on running scenarios.

### Example worker installs using Helm

Install the example worker as an OAM application with 10 replicas:
```
helm install psr-example manifests/charts/worker --set imageName=ghcr.io/verrazzano/psr-backend:local-582bfcfcf --replicas=10
```

Install the example worker as an OAM application using the default image in the helm chart, providing an imagePullSecret
```
helm install psr-example manifests/charts/worker --set imagePullSecrets[0].name=verrazzano-container-registry
```

Install the WriteLogs worker as a Kubernetes deployment using the default image in the helm chart.  Note the appType must be supplied:
```
helm install psr-writelogs manifests/charts/worker --set appType=k8s -f manifests/usecases/opensearch/writelogs.yaml
```

Install the WriteLogs worker as an OAM application with 5 worker threads and the default 1 replica.
```
helm install psr-writelogs manifests/charts/worker --set global.envVars.PSR_WORKER_THREAD_COUNT=5 -f manifests/usecases/opensearch/writelogs.yaml
```

## Workers
All of the workers are deployed using the same worker Helm chart. Worker configs are stored in global.envVars values.
There are common values that are available to each worker as follows:

```
global:
  envVars:
    PSR_WORKER_TYPE - type of worker
    default: ops-postlogs
    
    PSR_LOOP_SLEEP - duration to sleep between work iterations
    default: 1s

    PSR_NUM_LOOPS - number of iterations per worker thread
    default: -1 (run forever)
    
    PSR_WORKER_THREAD_COUNT - threads per worker
    default: 1
```

The following section describes each worker.

### Example
#### Description
The example worker periodically logs messages, it doesn't provide metrics.
   
#### Configuration
no configuration overrides
   
#### Run
```
helm install psr-example manifests/charts/worker 
```

### WriteLogs
#### Description
The WriteLogs worker periodically logs messages.  The goal is to put a load on OpenSearch since fluentd collects the container logs and
sends them to OpenSearch.

#### Configuration
no configuration overrides

#### Run
```
helm install psr-writelogs manifests/charts/worker -f manifests/usecases/opensearch/writelogs.yaml
```

### GetLogs
#### Description
The GetLogs worker periodically gets messages from OpenSearch, in-cluster.  This worker must run in the mesh.

#### Configuration
no configuration overrides

#### Run
```
helm install psr-getlogs manifests/charts/worker -f manifests/usecases/opensearch/getlogs.yaml -n istioEnabledNamespace
```

### PostLogs
#### Description
The PostLogs worker makes batch http post requests to OpenSearch, in-cluster. This worker must run in the mesh.

#### Configuration
```
global:
  envVars:
    LOG_ENTRIES - number of log messages per post request
    default: 1
    
    LOG_LENGTH - number of characters per log message
    default: 1
```

#### Run
```
helm install psr-postlogs manifests/charts/worker -f manifests/usecases/opensearch/postlogs.yaml  -n istioEnabledNamespace
```

### Scale
#### Description
The Scale worker continuously scales one tier of OpenSearch out and in. This worker must run in the mesh.

#### Configuration
```
global:
  envVars:
    OPEN_SEARCH_TIER - OpenSearch tier
    default: master
    
    MIN_REPLICA_COUNT - number of replicas to scale in to
    default: 3
    
    MAX_REPLICA_COUNT - number of replicas to scale out to
    default: 5
```

#### Run
```
helm install psr-scale manifests/charts/worker -f manifests/usecases/opensearch/scale.yaml -n istioEnabledNamespace
```

### Restart
#### Description
The restart worker continuously restarts pods one tier of OpenSearch.  This worker must run in the mesh.

#### Configuration
```
global:
  envVars:
    OPEN_SEARCH_TIER - OpenSearch tier
    default: none
```

#### Run
```
helm install psr-scale manifests/charts/worker -f manifests/usecases/opensearch/restart.yaml -n istioEnabledNamespace
```

### HTTPGet
#### Description
The HTTPGet worker makes http requests to the endpoint for a specified service.

#### Configuration
```
global:
  envVars:
    SERVICE_NAME - service name
    default: ""
    
    SERVICE_NAMESPACE - service namespace
    default: ""
    
    SERVICE_PORT - service port
    default: ""
    
    PATH - service path 
    default: ""
```

#### Run
```
helm install psr-httpget manifests/charts/worker -f manifests/usecases/http/get.yaml
```

## Developing scenarios and use cases
Refer to the [PSR developer guide](./DEVELOPER.md) to learn how to develop new use cases and scenarios.
