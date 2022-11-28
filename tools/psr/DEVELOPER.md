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

## Areas
PSR workers and scenarios are grouped into areas.  The following area names are used in the source code and YAML configuration.
They are not exposed in metrics names.

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


## Implementing a Worker
A worker is the code that implements a single use case.  Workers are organized per area, where each aread typically maps 
to a Verrazzano backend component, but that doesn't have to be the case.  You can see OpenSearch and HTTP workers
in the [workers](./backend/workers) package.

Lets use a new mysql worker that scales the MySQL database as an example in the following section.

Following are the steps to implement a worker:
1. Add a worker type named `WorkerTypeMysqlScale = mysql-scale` to [config.go](./backend/config/config.go)
2. Create a package named `mysql` in [workers](./backend/workers)
3. Create a go file name `scale.go`
4. Stub out the [worker SPI](./backend/spi/worker.go) SPI implementation in `scale.go`  You can copy the ops scale worker as a starting point.
5. Add your worker case to the switch statement in [manager.go](./backend/workmanager/manager.go)



