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
PSR workers and scenarios are grouped into areas.  The following area names are used in the source code and YAML configuration.
They are not exposed in metrics names, rather each `worker.go` file specifies the metrics prefix, which is the long name.  
For example, the OpenSearch worker uses the metric prefix `opensearch`

1. oam - oam applications, app operator
2. cm - cert-manager
3. cluster - cluster operator, multicluster
4. coh - Coherence
5. dns - external dns
6. jaeger - Jaeger
7. kc - Keycloak
8. http - HTTP tests
9. istio - Istio, Kiali
10. mysql - mysql
11. nginx - NGINX Ingress Controller, Authproxy
12. ops - OpenSearch, OpenSearchDashboards, Fluentd, VMO
13. prom - Prometheus stack, Kabana
14. rancher - Rancher
15. velero - Velero
16. wls - Weblogic


## Developing a worker
A worker is the code that implements a single use case.  Workers are organized per area, where each aread typically maps 
to a Verrazzano backend component, but that doesn't have to be the case.  You can see OpenSearch and HTTP workers
in the [workers](./backend/workers) package.

We will create a new mysql worker that queries the MySQL database as an example in the following section.

### Stubbing out a worker
Following are the first steps to implement a worker:
1. Add a worker type named `WorkerTypeMysqlScale = mysql-scale` to [config.go](./backend/config/config.go)
2. Create a package named `mysql` in package [workers](./backend/workers)
3. Create a file `query.go` in the `mysql` package and do the following:
   1. Stub out the [worker interface](./backend/spi/worker.go) implementation in `query.go`  You can copy the ops getlogs worker as a starting point.
   2. Change the const metrics prefix to `metricsPrefix = "mysql_query"`
   3. Rename the `NewGetLogsWorker` function to `NewQueryWorker`
   4. Change the `GetWorkerDesc` function to return information about the worker
   6. Change the DoWork function to  `fmt.Println("hello mysql query worker")`
4. Add your worker case to the switch statement in [manager.go](./backend/workmanager/manager.go)
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
2. Edit [usecases/mysql/query.yaml](./manifests/usecases/mysql/query.yaml) and add the following section
```
# activate subchart
mysql:
  enabled: true
```
3. You will need to install the chart in an Istio enabled namespace. 
4. Test the chart in an Verrazzano installation using the same Helm command as previously, but also specify the namespace:
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

