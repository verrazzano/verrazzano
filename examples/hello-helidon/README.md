
# Hello World Helidon Example Application

This example provides a simple *Hello World* REST service written with [Helidon](https://helidon.io)

### Install example

* Pre-requisites: Install Verrazzano following the [installation instructions](../install/README.md).

* Install example

```
    kubectl apply -f ./hello-world-model.yaml
    kubectl apply -f ./hello-world-binding.yaml
```

* Verify if all objects have started. Objects are started in the *greeting*, *verrazzano-system* and *monitoring*
  namespaces. The following code block shows the objects to expect. Objects not related to this sample application
  have been removed from the list.

```
    kubectl get all -n greet
    NAME                                          READY   STATUS    RESTARTS   AGE
    pod/hello-world-application-bb58ccfd6-2q9mb   3/3     Running   0          19m
    pod/hello-world-application-bb58ccfd6-w2zz5   3/3     Running   0          19m
    
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



* Get the External IP for istio-ingressgateway service

```
    kubectl get service istio-ingressgateway -n istio-system
```

* Use the external IP to call the different endpoints of the greeting REST service:
    - Default greeting message: `curl -X GET http://<external_ip>/greet`
    - Greet Robert: `curl -X GET http://<external_ip>/greet/Robert`

###Uninstall example

* Uninstall example

```
    kubectl delete -f ./hello-world-binding.yaml
    kubectl delete -f ./hello-world-model.yaml
```

