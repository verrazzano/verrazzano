
# Hello World Helidon

This example application provides a simple *Hello World* REST service written with [Helidon](https://helidon.io).
Check the [Verrazzano examples](https://github.com/verrazzano/examples) for more information and the code of this
application.

## Deploying the example application

1. Pre-requisites: Install Verrazzano following the [installation instructions](../install/README.md).

1. Run the following script to deploy the Verrazzano Model and Binding for the example application.

    ### Using an OKE Cluster

    ```
    ./install-hello-world.sh
    pod/verrazzano-operator-7c785bb84b-d7tlx condition met
    pod/verrazzano-admission-controller-58cf8b4b89-64vzk condition met
    verrazzanomodel.verrazzano.io/hello-world-model created
    verrazzanobinding.verrazzano.io/hello-world-binding created
    NAME                                      READY   STATUS            RESTARTS   AGE
    pod/hello-world-application-bb58ccfd6-6xmpg condition met
    pod/hello-world-application-bb58ccfd6-89ftc condition met
    {"message":"Hello World!"}
    ```

    ### Using a Kind Cluster

    ```
    export KIND_CLUSTER_NAME=verrazzano
    export CLUSTER_TYPE=KIND
    ./install-hello-world.sh
    pod/verrazzano-operator-66dff84cd7-v2jzs condition met
    pod/verrazzano-admission-controller-59dcbbdfdf-t828v condition met
    verrazzanomodel.verrazzano.io/hello-world-model created
    verrazzanobinding.verrazzano.io/hello-world-binding created
    NAME                                       READY   STATUS     RESTARTS   AGE
    pod/hello-world-application-868c5d9d88-qdsz2 condition met
    pod/hello-world-application-868c5d9d88-r5rbm condition met
    {"message":"Hello World!"}
    ```

   This script not only installs the model and binding, but also waits for the pods in the *greet* namespace to be
   ready, and then calls one of the endpoints provided by the REST service implemented by the example application. In the
   next sections we provide more details of the application and endpoints provided by it.

1. Verify if all objects have started. Objects are started in the *greet*, *verrazzano-system* and *monitoring*
  namespaces. The following code block shows the objects to expect. Objects not related to this sample application
  have been removed from the list.

    ```
    kubectl get all -n greet
    NAME                                          READY   STATUS    RESTARTS   AGE
    pod/hello-world-application-bb58ccfd6-6xmpg   3/3     Running   0          19m
    pod/hello-world-application-bb58ccfd6-89ftc   3/3     Running   0          19m

    NAME                              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
    service/hello-world-application   ClusterIP   10.96.119.252   <none>        8080/TCP   19m

    NAME                                      READY   UP-TO-DATE   AVAILABLE   AGE
    deployment.apps/hello-world-application   2/2     2            2           19m

    NAME                                                DESIRED   CURRENT   READY   AGE
    replicaset.apps/hello-world-application-bb58ccfd6   2         2         2       19m

    kubectl get all -n verrazzano-system
    NAME                                                        READY   STATUS    RESTARTS   AGE
    pod/vmi-hello-world-binding-api-7f74b6bcc4-8sqjm            1/1     Running   0          19m
    pod/vmi-hello-world-binding-es-data-0-7c98fd4fcf-sxgdp      2/2     Running   0          19m
    pod/vmi-hello-world-binding-es-data-1-788b454c5-2g7ct       2/2     Running   0          19m
    pod/vmi-hello-world-binding-es-ingest-676f89db-zkh4z        1/1     Running   0          19m
    pod/vmi-hello-world-binding-es-master-0                     1/1     Running   0          19m
    pod/vmi-hello-world-binding-es-master-1                     1/1     Running   0          19m
    pod/vmi-hello-world-binding-es-master-2                     1/1     Running   0          19m
    pod/vmi-hello-world-binding-grafana-5cc57fd5b9-xk5z2        1/1     Running   0          19m
    pod/vmi-hello-world-binding-kibana-8654ccd97-vkl48          1/1     Running   0          19m
    pod/vmi-hello-world-binding-prometheus-0-54fb4db4d7-hkcpr   3/3     Running   0          19m
    pod/vmi-hello-world-binding-prometheus-gw-9f6d54f5b-b887x   1/1     Running   0          19m

    NAME                                              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)             AGE
    service/vmi-hello-world-binding-api               ClusterIP   10.96.215.157   <none>        9097/TCP            19m
    service/vmi-hello-world-binding-es-data           ClusterIP   10.96.109.124   <none>        9100/TCP            19m
    service/vmi-hello-world-binding-es-ingest         ClusterIP   10.96.113.71    <none>        9200/TCP            19m
    service/vmi-hello-world-binding-es-master         ClusterIP   None            <none>        9300/TCP            19m
    service/vmi-hello-world-binding-grafana           ClusterIP   10.96.149.121   <none>        3000/TCP            19m
    service/vmi-hello-world-binding-kibana            ClusterIP   10.96.4.240     <none>        5601/TCP            19m
    service/vmi-hello-world-binding-prometheus        ClusterIP   10.96.136.127   <none>        9090/TCP,9100/TCP   19m
    service/vmi-hello-world-binding-prometheus-gw     ClusterIP   10.96.158.80    <none>        9091/TCP            19m

    NAME                                                    READY   UP-TO-DATE   AVAILABLE   AGE
    deployment.apps/vmi-hello-world-binding-api             1/1     1            1           19m
    deployment.apps/vmi-hello-world-binding-es-data-0       1/1     1            1           19m
    deployment.apps/vmi-hello-world-binding-es-data-1       1/1     1            1           19m
    deployment.apps/vmi-hello-world-binding-es-ingest       1/1     1            1           19m
    deployment.apps/vmi-hello-world-binding-grafana         1/1     1            1           19m
    deployment.apps/vmi-hello-world-binding-kibana          1/1     1            1           19m
    deployment.apps/vmi-hello-world-binding-prometheus-0    1/1     1            1           19m
    deployment.apps/vmi-hello-world-binding-prometheus-gw   1/1     1            1           19m

    NAME                                                              DESIRED   CURRENT   READY   AGE
    replicaset.apps/vmi-hello-world-binding-api-7f74b6bcc4            1         1         1       19m
    replicaset.apps/vmi-hello-world-binding-es-data-0-7c98fd4fcf      1         1         1       19m
    replicaset.apps/vmi-hello-world-binding-es-data-1-788b454c5       1         1         1       19m
    replicaset.apps/vmi-hello-world-binding-es-ingest-676f89db        1         1         1       19m
    replicaset.apps/vmi-hello-world-binding-grafana-5cc57fd5b9        1         1         1       19m
    replicaset.apps/vmi-hello-world-binding-kibana-8654ccd97          1         1         1       19m
    replicaset.apps/vmi-hello-world-binding-prometheus-0-54fb4db4d7   1         1         1       19m
    replicaset.apps/vmi-hello-world-binding-prometheus-gw-9f6d54f5b   1         1         1       19m

    NAME                                                 READY   AGE
    statefulset.apps/vmi-hello-world-binding-es-master   3/3     19m

    kubectl get all -n monitoring
    NAME                                                   READY   STATUS    RESTARTS   AGE
    pod/prom-pusher-hello-world-binding-787d9c6894-62nts   1/1     Running   0          19m

    NAME                                              READY   UP-TO-DATE   AVAILABLE   AGE
    deployment.apps/prom-pusher-hello-world-binding   1/1     1            1           19m

    NAME                                                         DESIRED   CURRENT   READY   AGE
    replicaset.apps/prom-pusher-hello-world-binding-787d9c6894   1         1         1       19m
    ```
## Explore the example application

The Hello World Helidon example application implements a REST service with the following endpoints:

- `/greet` - Returns a default greeting message that is stored in memory in an application scoped bean.
This endpoint accepts `GET` HTTP request method.
- `/greet/{name}` - Returns a greeting message including the name provided in the path parameter. This
endpoint accepts `GET` HTTP request method.
- `/greet/greeting` - Changes the greeting message to be used in future calls to the other endpoints. This
endpoint accepts `PUT` HTTP request method, and a json payload.

The steps to test these endpoints are described next.

1. Get the IP address and port to call the REST service.
    ### Using OKE cluster
    Get the EXTERNAL-IP for istio-ingressgateway service:

    ```
    SERVER=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq -r '.status.loadBalancer.ingress[0].ip')
    PORT=80
    ```

   ### Using Kind Cluster
   Get the IP of one node of the cluster and the port from the istio-ingressgateway service:

   ```
   SERVER=$(kubectl get node ${KIND_CLUSTER_NAME}-worker -o json | jq -r '.status.addresses[] | select (.type == "InternalIP") | .address')
   PORT=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq '.spec.ports[] | select(.port == 80) | .nodePort')
   ```

1. Use the IP address and port to call the different endpoints of the greeting REST service:

    ```
    # Get default message
    curl -s -X GET http://"${SERVER}":"${PORT}"/greet
    {"message":"Hello World!"}

    # Get message for Robert:
    curl -s -X GET http://"${SERVER}":"${PORT}"/greet/Robert
    {"message":"Hello Robert!"}

    # Change the message:
    curl -s -X PUT -H "Content-Type: application/json" -d '{"greeting" : "Hallo"}' http://"${SERVER}":"${PORT}"/greet/greeting

    # Get message for Robert again:
    $ curl -s -X GET http://"${SERVER}":"${PORT}"/greet/Robert
    {"message":"Hallo Robert!"}
    ```

## Uninstalling the example application

1. Run the following script to delete the Verrazzano Model and Binding for the example application:

    ```
    ./uninstall-hello-world.sh
    verrazzanobinding.verrazzano.io "hello-world-binding" deleted
    verrazzanomodel.verrazzano.io "hello-world-model" deleted
    ```

