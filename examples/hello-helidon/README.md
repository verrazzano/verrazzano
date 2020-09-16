<!--
Copyright (c) 2020, Oracle and/or its affiliates.
Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
-->
# Hello World Helidon

This guide demonstrates deploying a *Hello World* REST service to [Verrazzano](https://verrazzano.io/).
The example application is written using [Helidon](https://helidon.io). 
Implementation details can be found in the [Helidon tutorial](https://helidon.io/docs/latest/#/mp/guides/10_mp-tutorial).
Check the [Verrazzano examples](https://github.com/verrazzano/examples/tree/master/hello-helidon) repository for the application's source code.

This guide should take about 10 minutes to complete.

## Prerequisites

* An existing Verrazzano environment is required for using this guide.
  If required, install Verrazzano using the [installation instructions](../../install/README.md).  

## Overview

Deploying an application to Verrazzano requires two primary steps.
0. Applying the application's Verrazzano model resource.
0. Applying the application's Verrazzano binding resource.

Details about applying these resources and validating the results are provided in the remainder of this guide.

### Verrazzano Model

A Verrazzano model is a 
[Kubernetes Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
describing an application's general composition and environmental requirements.
The Verrazzano model for this guide is shown below.
This model describes an application which is implemented by a specific docker image and has a single endpoint.
More details about Verrazzano models can be found in the [Verrazzano model documentation](https://verrazzano.io/docs/reference/model/).

```yaml
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
apiVersion: verrazzano.io/v1beta1
kind: VerrazzanoModel
metadata:
  name: hello-world-model
  namespace: default
spec:
  description: "Hello World application"
  helidonApplications:
    - name: "hello-world-application"
      image: "container-registry.oracle.com/verrazzano/example-hello-world-helidon:0.1.10-3-e5ae893-124"
      connections:
        - ingress:
            - name: "greet-ingress"
              match:
                - uri:
                    prefix: "/greet"
```

This table provides a brief description of each field in a model.

| Field                                                           | Description                                                        |
|-----------------------------------------------------------------|--------------------------------------------------------------------|
| `apiVersion`                                                    | Version of the Verrazzano model custom resource definition         |
| `kind`                                                          | Standard name of the Verrazzano model custom resource definition   |
| `metadata.name`                                                 | Name of the model's Kubernetes custom resource                     |
| `metadata.namespace`                                            | Namespace of the model's Kubernetes custom resource                |
| `spec.helidonApplications.name`                                 | Name of the application                                            |
| `spec.helidonApplications.image`                                | Docker image for the application's implementation                  |
| `spec.helidonApplications.connections.ingress.name`             | Logical name of an application ingress                             |
| `spec.helidonApplications.connections.ingress.match.uri.prefix` | Physical URI prefix for the application's ingress                  |
    
### Verrazzano Binding

A Verrazzano binding is a 
[Kubernetes Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
which provides environment specific customizations.
The Verrazzano binding for this guide shown below.
This binding specifies that the application be placed in the 'local' cluster 
within the 'greet' namespace having an ingress endpoint bound to DNS name '*'. 
More details about Verrazzano bindings can be found in the Verrazzano 
[binding documentation](https://verrazzano.io/docs/reference/binding/).

```yaml
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
apiVersion: verrazzano.io/v1beta1
kind: VerrazzanoBinding
metadata:
  name: hello-world-binding
  namespace: default
spec:
  description: "Hello World Application binding"
  modelName: hello-world-model
  placement:
    - name: local
      namespaces:
        - name: greet
          components:
            - name: hello-world-application
  ingressBindings:
    - name: "greet-ingress"
      dnsName: "*"
```

This table provides a brief description of each field in the model.

| Field                                       | Description                                                             |
|---------------------------------------------|-------------------------------------------------------------------------|
| `apiVersion`                                | Version of the Verrazzano binding custom resource definition            |
| `kind`                                      | Standard name of the Verrazzano binding custom resource definition      |
| `metadata.name`                             | Name of the binding's Kubernetes custom resource                        |
| `metadata.namespace`                        | Namespace of the binding's Kubernetes custom resource resource          |
| `spec.modelName`                            | Reference to the application's Verrazzano model custom resource         |
| `spec.placement.name`                       | Name of the Kubernetes cluster as defined in the Verrazzano environment |
| `spec.placement.namespaces.name`            | Name of the namespace in which to place the components                  |
| `spec.placement.namespaces.components.name` | Name of the component within the namespace                              |
| `spec.ingressBindings.name`                 | Reference to an ingress from the model                                  |
| `spec.ingressBineings.dnsName`              | The DNS name to associate with the ingress                              |

## Deploy the application

Below are the steps required to deploy the example application.
Steps similar to the `apply` steps below would be used to deploy any application to Verrazzano.

0. Clone the Verrazzano [repository](https://github.com/verrazzano/verrazzano).

   ```bash
   git clone https://github.com/verrazzano/verrazzano.git
   ```

0. Change the current directory to the example hello-helidon directory.

   ```bash
   cd verrazzano/examples/hello-helidon
   ``` 

   _Note: The remainder of this guide uses file locations relative to this directory._ 

0. Apply the [application's Verrazzano model](../../examples/hello-helidon/hello-world-model.yaml).

   ```bash
   kubectl apply -f ./hello-world-model.yaml
   ```

   This step causes the validation and registration of the model resource.
   No other resources or objects are created as a result.
   Bindings applied in the future may reference such registered model.

0. Apply the [application's Verrazzano binding](../../examples/hello-helidon/hello-world-binding.yaml).

   ```bash
   kubectl apply -f ./hello-world-binding.yaml
   ```
   
   This step causes the validation and then registration of the binding.
   The binding registration triggers the activation of a number of Verrazzano operators.
   These operators create Kubernetes objects (e.g. deployments, replacsets, pods, services, ingresses) 
   that collectively provide and support the application.
   
_Note: The steps above that apply the model and binding are included in the convenience script 
[install-hello-world.sh](../../examples/hello-helidon/install-hello-world.sh)._
   
## Verify the application deployment

  Applying the binding initiates the creation of several Kubernetes objects.
  Actual creation and initialization of these object occurs asynchronously.
  The following steps provide commands for determining these objects are ready for use.
  
  _Note: Verification steps similar to those that follow are included in the convenience script 
  [install-hello-world.sh](../../examples/hello-helidon/install-hello-world.sh)._ 
      
  _Note: Many other Kubernetes objects unrelated to the example application may also exist.  
  Those have been omitted from the lists below._
  
0. Verify the Helidon application pod is running  
   
   ```bash
   kubectl get pods -n greet | grep hello-world-application

   hello-world-application-648f8f79d9-8xkhl   3/3     Running   0          20h
   ```
   
   The table below is an example of the Kubernetes objects that might be created to implement a Helidon application.

   |Namespace        |Name                                                  |Kind       |
   |-----------------|------------------------------------------------------|-----------|
   |greet            |hello-world-application                               |Deployment |
   |greet            |hello-world-application                               |Service    |
   |greet            |hello-world-application-648f8f79d9                    |ReplicaSet |
   |greet            |hello-world-application-648f8f79d9-8xkhl              |Pod        |

0. Verify the Verrazzano Helidon application operator pod is running

   ```bash
   kubectl get pods -n verrazzano-system | grep verrazzano-helidon-app-operator

   verrazzano-helidon-app-operator-d746d7bc6-67th8          1/1     Running   0          20h
   ```
     
   A single verrazzano-helidon-app-operator manages the lifecycle of all Helidon based applications within the cluster.

   The table below is an example of the Kubernetes objects that might be created to managing a Helidon application's lifecycle.
   
   |Namespace        |Name                                                  |Kind       |
   |-----------------|------------------------------------------------------|-----------|
   |verrazzano-system|verrazzano-helidon-app-operator                       |Deployment |
   |verrazzano-system|verrazzano-helidon-app-operator-d746d7bc6             |ReplicaSet |
   |verrazzano-system|verrazzano-helidon-app-operator-d746d7bc6-67th8       |Pod        |
   |verrazzano-system|verrazzano-helidon-app-operator-metrics               |Service    |

0. Verify the Verrazzano monitoring infrastructure is running

   ```bash
   kubectl get pods -n verrazzano-system | grep vmi-hello-world-binding

   vmi-hello-world-binding-api-69987d6dbb-stpd4             1/1     Running   0          21h
   vmi-hello-world-binding-es-data-0-55b679d6bb-5g2bf       2/2     Running   0          21h
   vmi-hello-world-binding-es-data-1-7888dbdfcf-76ff9       2/2     Running   0          21h
   vmi-hello-world-binding-es-ingest-b7d59fb69-6hbr4        1/1     Running   0          21h
   vmi-hello-world-binding-es-master-0                      1/1     Running   0          21h
   vmi-hello-world-binding-es-master-1                      1/1     Running   0          21h
   vmi-hello-world-binding-es-master-2                      1/1     Running   0          21h
   vmi-hello-world-binding-grafana-85b669cdbc-4rszf         1/1     Running   0          21h
   vmi-hello-world-binding-kibana-64f958c7f-knxk4           1/1     Running   0          21h
   vmi-hello-world-binding-prometheus-0-598f79557-ttzmz     3/3     Running   0          21h
   vmi-hello-world-binding-prometheus-gw-6df8bf4689-dmfxh   1/1     Running   0          21h
   ```

   The table below is an example of the Kubernetes objects that might be created for application monitoring.

   |Namespace        |Name                                                  |Kind       |
   |-----------------|------------------------------------------------------|-----------|
   |verrazzano-system|vmi-hello-world-binding-api                           |Deployment |
   |verrazzano-system|vmi-hello-world-binding-api                           |Service    |
   |verrazzano-system|vmi-hello-world-binding-api-69987d6dbb                |ReplicaSet |
   |verrazzano-system|vmi-hello-world-binding-api-69987d6dbb-stpd4          |Pod        |
   |verrazzano-system|vmi-hello-world-binding-es-data                       |Service    |
   |verrazzano-system|vmi-hello-world-binding-es-data-0                     |Deployment |
   |verrazzano-system|vmi-hello-world-binding-es-data-0-55b679d6bb          |ReplicaSet |
   |verrazzano-system|vmi-hello-world-binding-es-data-0-55b679d6bb-5g2bf    |Pod        |
   |verrazzano-system|vmi-hello-world-binding-es-data-1                     |Deployment |
   |verrazzano-system|vmi-hello-world-binding-es-data-1-7888dbdfcf          |ReplicaSet |
   |verrazzano-system|vmi-hello-world-binding-es-data-1-7888dbdfcf-76ff9    |Pod        |
   |verrazzano-system|vmi-hello-world-binding-es-ingest                     |Deployment |
   |verrazzano-system|vmi-hello-world-binding-es-ingest                     |Service    |
   |verrazzano-system|vmi-hello-world-binding-es-ingest-b7d59fb69           |ReplicaSet |
   |verrazzano-system|vmi-hello-world-binding-es-ingest-b7d59fb69-6hbr4     |Pod        |
   |verrazzano-system|vmi-hello-world-binding-es-master                     |Service    |
   |verrazzano-system|vmi-hello-world-binding-es-master                     |StatefulSet|
   |verrazzano-system|vmi-hello-world-binding-es-master-0                   |Pod        |
   |verrazzano-system|vmi-hello-world-binding-es-master-1                   |Pod        |
   |verrazzano-system|vmi-hello-world-binding-es-master-2                   |Pod        |
   |verrazzano-system|vmi-hello-world-binding-grafana                       |Deployment |
   |verrazzano-system|vmi-hello-world-binding-grafana                       |Service    |
   |verrazzano-system|vmi-hello-world-binding-grafana-85b669cdbc            |ReplicaSet |
   |verrazzano-system|vmi-hello-world-binding-grafana-85b669cdbc-4rszf      |Pod        |
   |verrazzano-system|vmi-hello-world-binding-kibana                        |Deployment |
   |verrazzano-system|vmi-hello-world-binding-kibana                        |Service    |
   |verrazzano-system|vmi-hello-world-binding-kibana-64f958c7f              |ReplicaSet |
   |verrazzano-system|vmi-hello-world-binding-kibana-64f958c7f-knxk4        |Pod        |
   |verrazzano-system|vmi-hello-world-binding-prometheus                    |Service    |
   |verrazzano-system|vmi-hello-world-binding-prometheus-0                  |Deployment |
   |verrazzano-system|vmi-hello-world-binding-prometheus-0-598f79557        |ReplicaSet |
   |verrazzano-system|vmi-hello-world-binding-prometheus-0-598f79557-ttzmz  |Pod        |
   |verrazzano-system|vmi-hello-world-binding-prometheus-gw                 |Deployment |
   |verrazzano-system|vmi-hello-world-binding-prometheus-gw                 |Service    |
   |verrazzano-system|vmi-hello-world-binding-prometheus-gw-6df8bf4689      |ReplicaSet |
   |verrazzano-system|vmi-hello-world-binding-prometheus-gw-6df8bf4689-dmfxh|Pod        |

0. Verify the Verrazzano metrics collection infrastructure is running

   ```bash
   kubectl get pods -n monitoring | grep prom-pusher-hello-world-binding

   prom-pusher-hello-world-binding-6648484f89-t8rf8   1/1     Running   0          21h
   ```

   The table below is an example list of Kubernetes objects that should have been created for application metrics collection.

   |Namespace        |Name                                                  |Kind       |
   |-----------------|------------------------------------------------------|-----------|
   |monitoring       |prom-pusher-hello-world-binding                       |Deployment |
   |monitoring       |prom-pusher-hello-world-binding-6648484f89            |ReplicaSet |
   |monitoring       |prom-pusher-hello-world-binding-6648484f89-t8rf8      |Pod        |
  
0. Diagnose failures

   View the event logs of any pod not entering the "Running" state within a reasonable length of time.
   
   ```bash
   kubectl describe pod -n greet hello-world-application-648f8f79d9-8xkhl
   ``` 

## Explore the application's functionality

The example application implements a REST service with the following endpoints:
- `/greet` - Returns a default greeting message that is stored in memory in an application-scoped bean.
This endpoint accepts the `GET` HTTP request method.
- `/greet/{name}` - Returns a greeting message including the name provided in the path parameter. This
endpoint accepts the `GET` HTTP request method.
- `/greet/greeting` - Changes the greeting message to be used in future calls to the other endpoints. This
endpoint accepts the `PUT` HTTP request method, and a JSON payload.

Follow these steps to explore the application's functionality.

0.  Get the IP address of the load balancer exposing the applications REST service endpoints
    ```
    SERVER=$(kubectl get service -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}') && echo $SERVER
    ```

0.  Get the default message
    ```bash
    curl -s -X GET http://${SERVER}/greet

    {"message":"Hello World!"}
    ```

0.  Get a message for Robert
    ```bash
    curl -s -X GET http://${SERVER}/greet/Robert

    {"message":"Hello Robert!"}
    ```

0.  Update the default greeting
    ```bash
    curl -s -X PUT -H "Content-Type: application/json" -d '{"greeting" : "Greetings"}' http://${SERVER}/greet/greeting
    ```

0.  Get the new message for Robert
    ```bash
    curl -s -X GET http://${SERVER}/greet/Robert

    {"message":"Welcome Robert!"}
    ```

### Access the application's logs

Applications deployed using Verrazzano bindings automatically have log collection enabled.
These logs are collected using ElasticSearch and can be accessed using Kibana.

The URL used to access Kibana can be determined using the following commands.
 ```bash
KIBANA_HOST=$(kubectl get ingress -n verrazzano-system vmi-hello-world-binding-kibana -o jsonpath='{.spec.rules[0].host}')
KIBANA_URL="https://${KIBANA_HOST}"
echo "${KIBANA_URL}"
```

The username used to access Kibana is currently defaulted during Verrazzano install to `verrazzano`. 

The password used to access Kibana can be determined using the following commands.
```bash
echo $(kubectl get secret -n verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode)
``` 
 
### Access the application's metrics

Applications deployed using Verrazzano bindings automatically have metric collection enabled.
These metrics are collected using ElasticSearch and can be accessed using Grafana.

The URL used to access Grafana can be determined using the following commands.
```bash
GRAFANA_HOST=$(kubectl get ingress -n verrazzano-system vmi-hello-world-binding-grafana -o jsonpath='{.spec.rules[0].host}')
GRAFANA_URL="https://${GRAFANA_HOST}"
echo "${GRAFANA_URL}"
```

The username used to access Grafana is currently defaulted during Verrazzano install to `verrazzano`.
 
The password used to access Grafana can be determined using the following commands.
```bash
echo $(kubectl get secret -n verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode)
``` 

Alternativly metrics can be accessed directly using Prometheus.
The URL used for this access can be determined using the following commands.
```bash
PROMETHEUS_HOST=$(kubectl get ingress -n verrazzano-system vmi-hello-world-binding-prometheus -o jsonpath='{.spec.rules[0].host}')
PROMETHEUS_URL="https://${PROMETHEUS_HOST}"
echo "${PROMETHEUS_URL}"
```

The username and password used for Prometheus access are the same as for Grafana.

## Uninstall the application

Run the following commands to delete the application's Verrazzano binding and optionally Verrazzano model.

0. Delete the application's binding

   ```bash
   kubectl delete -f ./hello-world-binding.yaml
   ```
   
   The deletion of of the application's model will result in the destruction 
   of all application specific Kubernetes objects.
    
0. Delete the application's model (optional)

   ```bash
   kubectl delete -f ./hello-world-model.yaml
   ```
   _Note: This step is not required if other bindings for this application will be applied in the future._

The convenience script [uninstall-hello-world.sh](../../examples/uninstall-hello-world.sh) can be used to perform the steps above.