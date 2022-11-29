# PSR Developer Guide

This document describes how to develop PSR workers and scenarios that can be used to test individual 
Verrazzano components, or Verrazzano as a whole. Here is a summary of the steps you should follow, details will follow.

1. Get familiar with the PSR tool, run the example and some scenarios.
2. Decide what component you want to test.
3. Decide what you want to test for your first scenario.
4. Decide what workers you need to implement your scenario use cases.
5. Implement a single worker and test it using Helm.
6. Create or update a scenario that includes the worker.
7. Test the scenario using the PSR CLI (psrctl).
8. Repeat steps 5-7 until the scenario is complete.
9. Update the README with your worker information.

## Prerequisites
- Read the [Verrazzano PSR README](./README.md)  to get familiar with the PSR concepts and structure of the source code.
- A Kubernetes cluster with Verrazzano installed (full installation or the components you are testing).

## PSR Areas
Workers are organized per area, where each aread typically maps to a Verrazzano backend component, but that isn't always 
the case.  You can see OpenSearch and HTTP workers in the [workers](./backend/workers) package.PSR workers and scenarios 
are grouped into areas.  

The following area names are used in the source code and YAML configuration.
They are not exposed in metrics names, rather each `worker.go` file specifies the metrics prefix, which is the long name.  
For example, the OpenSearch worker uses the metric prefix `opensearch`

1. argo - Argo
2. oam - OAM applications, Verrazzano application operator
3. cm - cert-manager
4. cluster - Verrazzano Cluster operator, multicluster
5. coh - Coherence
6. dns - ExternalDNS
7. jaeger - Jaeger
8. kc - Keycloak
9. http - HTTP tests
10. istio - Istio, Kiali
11. mysql - MySQL
12. nginx - NGINX Ingress Controller, AuthProxy
13. ops - OpenSearch, OpenSearchDashboards, Fluentd, VMO
14. prom - Prometheus stack, Kabana
15. rancher - Rancher
16. velero - Velero
17. wls - Weblogic


## Developing a worker
As mentioned in the README, a worker is the code that implements a single use case. For example, a worker might continuously
scale OpenSearch in and out.  The `DoWork` function is the code that actually does the work and is called repeatedly by the
`runner`.  DoWork does whatever it needs to do to perform work, this include blocking as needed waiting for conditions.

### Worker Tips
Here is some important information to know about workers, much of it is repeated in the README.

1. Worker code runs in a backend pod
2. The same backend pod has the code for all the workers, but only one worker is executing
3. Workers can have multiple threads doing work (scale up)
4. Workers can have multiple replicas (scale out)
5. Workers are configured using environment variables 
6. Workers should only do one thing (e.g. scale a component in and out)
7. All worker should emit metrics
8. Workers must wait for their dependencies before doing work (e.g. Verrazzano ready)
9. Worker `DoWork` function is called repeatedly in a loop by the `runner`
10. Some workers must be run in an Istio enabled namespace (depends on what the worker does)
11. A Worker might need additional Kubernetes resources to be created (e.g. AuthorizationPolicies)
12. Workers can be run as Kubernetes deployments or OAM apps (default), this is specified at Helm install
13. All workers run as cluster-admin

### Worker Chart and Overrides
Workers are deployed using Helm where there is a single Helm chart for all workers along with area specific Helm subcharts.
Each worker specifies the value overrides in a YAML file, such as the environment variables needed to configure
worker. If an area specific subchart is needed then it must be enabled in the override file.

The overrides YAML file is in manifests/usecases/<area>/<worker>.yaml.
For example, [usecases/opensearch/getlogs.yaml](./manifests/usecases/opensearch/getlogs.yaml)

```
global:
  envVars:
    PSR_WORKER_TYPE: ops-getlogs
    
# activate subchart
opensearch:
  enabled: true

```

### Sample MySQL worker
To make this section easier to follow, we will describe creating a new mysql worker that queries the MySQL database.  
In general, when creating a worker, it is easiest to just copy an existing worker that does the same type of action (e.g. scale) 
and modify it as needed for your component.  When it makes sense, common code should be factored out and reused by multiple workers.

### Stubbing-out a worker
Following are the first steps to implement a worker:

1. Add a worker type named `WorkerTypeMysqlScale = mysql-scale` to [config.go](./backend/config/config.go)
2. Create a package named `mysql` in package [workers](./backend/workers)
3. Create a file `query.go` in the `mysql` package and do the following:
   1. Stub out the [worker interface](./backend/spi/worker.go) implementation in `query.go`  You can copy the ops getlogs worker as a starting point.
   2. Change the const metrics prefix to `metricsPrefix = "mysql_query"`
   3. Rename the `NewGetLogsWorker` function to `NewQueryWorker`
   4. Change the `GetWorkerDesc` function to return information about the worker
   6. Change the DoWork function to  `fmt.Println("hello mysql query worker")`
4. Add your worker case to the `getWorker` function in [manager.go](./backend/workmanager/manager.go)
5. Add a directory named `mysql` to [usecases](./manifests/usecases)
6. Copy [usecases/opensearch/getlogs.yaml](./manifests/usecases/opensearch/getlogs.yaml) to a file named `usecases/mysql/query.yaml` 
7. Edit query.yaml  
   1. change `PSR_WORKER_TYPE: ops-getlogs` to `PSR_WORKER_TYPE: mysql-query` 
   2. remove the opensearch-authpol section

### Testing a stubbed-out worker
This section shows how to test the new worker in a Kind cluster. 

1. Test the example worker first by building the image, loading it into the cluster and running the example worker
   1. `make run-example-k8s`
   2. take note of the image name:tag that is used with the --set override, for example the output might show this:
      1. helm upgrade --install psr manifests/charts/worker --set appType=k8s --set imageName=ghcr.io/verrazzano/psr-backend:local-4210a50
2. kubectl get pods to see the example worker, look at the pod logs to make sure it is logging
3. Delete the example worker
   1. `helm delete psr`
4. Run the mysql worker with the newly built image, an example image tag is shown below:
   1. `helm install psr manifests/charts/worker -f manifests/usescases/mysql/query.yaml --set appType=k8s --set imageName=ghcr.io/verrazzano/psr-backend:local-4210a50`
5. Look at the PSR mysql worker pod and make sure that it is logging `hello mysql query worker`
6. Delete the mysql worker
   1. `helm delete psr`


### Add worker specific charts
To function properly, certain workers need additional Kubernetes resources to be created.  Rather than having the worker create the 
resources at runtime, you can use a subchart to create them. The subchar will be shared by all workers in an area.  
Since the mysql query worker needs to access mysql directly within the cluster, it will need an Istio AuthorizationPolicy, 
just like the OpenSearch workers do.  This section will show how to add the chart and use it in the usecase YAML file.

1. Create a new subchart called mysql
   1. copy the opensearch chart from [manifests/charts/worker/charts/opensearch](./manifests/charts/worker/charts/opensearch-authpol) to [manifests/charts/worker/charts/mysql](./manifests/charts/worker/charts/mysql) 
   2. create the authorizationpolicy.yaml file with the correct policy to access mysql.  
   3. Delete the existing opensearch policy yaml files
   4. Update Chart.yaml and values.yaml to reflect mysql changes
2. Modify the [worker Chart.yaml](./manifests/charts/worker/Chart.yaml) file and add a depenedency for the mysql chart
```
dependencies:
  - name: mysql
    repository: file://../mysql
    version: 0.1.0
    condition: mysql.enabled
```
3. Edit [usecases/mysql/query.yaml](./manifests/usecases/mysql/query.yaml) and add the following section
```
# activate subchart
mysql:
  enabled: true
```
4. You will need to install the chart in an Istio enabled namespace. 
5. Test the chart in an Verrazzano installation using the same Helm command as previously, but also specify the namespace:
   1. `helm install psr manifests/charts/worker -n myns -f manifests/usescases/mysql/query.yaml --set appType=k8s --set imageName=ghcr.io/verrazzano/psr-backend:local-4210a50`

### Add metrics to worker
Worker metrics are very important and how we track the progress of a worker.  Before implmenting the `DoWork` and `PreconditionsMet` funcitons,
you should get metrics working.  The reason is that you will be able to easily test your metrics by running your worker in an IDE, 
then opening up your browser to http://localhost:9090/metrics.  Once you implement the real worker code (`DoWork`), you might need
to run in an Istio enabled namespace and will need to use prometheus or grafana to see the metrics.

The [runner](./backend/workmanager/runner.go) also emits metrics such as loop count, so you don't need to emit the same metrics.

1. Modify the `workerMetrics` struct to add the metrics that the worker will emit
2. Modify the `NewQueryWorker` funciton to specify the metrics descriptors.
   1. Use a CounterValue metric if the value can never go down, otherwise use GaugeValue or some other metric type.
   2. Don't specify the worker type prefix in the name field, that is automatically added to the metric name.
3. Modify the `GetMetricList` function returning the list of metrics
4. Modify DoWork to update the metrics as work is done. 
   1. You might have some metrics that you cannot implement until the full DoWork code is done. 
   2. Metric access must be thread-safe, use the atomic package like the other worker.
5. Test the worker using the Helm chart
6. Access the prometheus console and query the metrics.


### Implement the remainder of the worker code
Implement the remaining worker code in `query.go`, specifically `PreconditionsMet` abd `DoWork` Note that the query worker 
doesn't need a Kubernetes client since it knows the MySQL service name. If your worker needs
to call the Kubernetes API server, then use the [k8sclient](./backend/pkg/k8sclient) package.  See how the
OpenSearch [getlogs](./backend/workers/opensearch/scale/scale.go) worker uses ` k8sclient.NewPsrClient`.

1. Implement NewQueryWorker to create the worker instance
2. Change function GetEnvDescList to return configuration environment variables that the worker needs
   1. See the OpenSearch [getlogs](./backend/workers/opensearch/scale/scale.go) worker as an example
3. Implement DoWork. This method should not log, but if it really needs to log, then use the throttled Verrazzano logging,
   such as Progress or ErrorfThrottled
4. Test the worker using the Helm chart

**NOTE** The same worker struct and metrics is shared across all worker threads.  There is currently no state per worker.  Workers
that keep state, such as the scaling worker, normally only run in a single thread. 


## Creating and running scenarios
A scenarios is a collection of use cases with a curated configuration, that are run concurrently.  Typically, 
you should restrict a scenario use cases to a single area, but that is not a strict requirement.  You can run multiple 
scenarios concurrently so creating a mixed-area scenario might not be necessary.  If you do decide to create a mixed area scenario, 
then create it in a directory called scenario/mixed.

## PSR CLI (psrctl)
Scenarios are run by the PSR command line interface, `psrctl`.  The source code [manifests](./manifests) directory contains all
of the helm charts, use cases overrides, and scenario files.  These manifests files are built into the psrctl binary and accessed
internally at runtime, so the psrctl binary is self contained and there is no need for the user to provide external files.  However,
you can override the scenario directory at runtime with the `-d` flag.  This allows you to modify and test scenarions without having
to rebuild `psrctl`.  See `psrctl` help for details.

### PSR CLI Backend Image
If you build `psrctl` using make, the image tag is derived from the last commit id.  If that image has not been uploaded to
ghcr.io, you will need to run `make docker-push`.  Since that image is private will need to provide a secret with the ghcr.io
credentials with `psrctl -p`.  If you want to override the image name, use `psrctl -w`.

If you want to use a local image and load it into a kind cluster, they run `make kind-load-image` and specify that image
using `psrctl -w`.  This is the easiest way to develop and test a worker on Kind.

### Scenario files
Scenarios are specified by a scenario.yaml file along with usecase override files, one for each use case.
By convention, the files must be in the [scenarios](./manifests/scenarios) directory structured as follows:
```
<area>/<scenario-name>/scenario.yaml
<area>/<scenario-name>usecase-overrides/*
```
For example, the scenario to restart all OpenSearch tiers is in [restart-all-tiers](./manifests/scenarios/opensearch/restart-all-tiers)

### Scenario location




