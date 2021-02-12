# ToDo List

ToDo List is an example application containing a WebLogic component.
For more information and the source code of this application, see the [Verrazzano examples](https://github.com/verrazzano/examples).

## Prerequisites

* Install Verrazzano following the [installation instructions](https://verrazzano.io/docs/setup/install/installation/).
* In order to be able to download the example image, you must first accept the license agreement.
  * In a browser navigate to https://container-registry.oracle.com/ and sign in.
  * Search for `example-todo` and select the image name in the results.
  * Click Continue, then read and accept the license agreement.

The ToDo List application deployment artifacts are contained in the Verrazzano project located at
`<VERRAZZANO_HOME>/examples/todo-list`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.\
**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/todo-list`.

## Deploy the example application

1. Create a namespace for the ToDo List example and add a label identifying the namespace as managed by Verrazzano.
   ```
   kubectl create namespace todo-list
   kubectl label namespace todo-list verrazzano-managed=true
   ```

1. Create a `docker-registry` secret to enable pulling the ToDo List example image from the registry.
   ```
   kubectl create secret docker-registry tododomain-repo-credentials \
           --docker-server=container-registry.oracle.com \
           --docker-username=YOUR_REGISTRY_USERNAME \
           --docker-password=YOUR_REGISTRY_PASSWORD \
           --docker-email=YOUR_REGISTRY_EMAIL \
           -n todo-list
   ```

   Replace `YOUR_REGISTRY_USERNAME`, `YOUR_REGISTRY_PASSWORD`, and `YOUR_REGISTRY_EMAIL`
   with the values you use to access the registry.  

1. Create and label secrets for the WebLogic domain:
   ```
   kubectl create secret generic tododomain-weblogic-credentials \
     --from-literal=password=welcome1 \
     --from-literal=username=weblogic -n todo-list

   kubectl create secret generic tododomain-jdbc-tododb \
     --from-literal=password=welcome1 \
     --from-literal=username=derek -n todo-list

   kubectl -n todo-list label secret tododomain-jdbc-tododb weblogic.domainUID=tododomain

   kubectl create secret generic tododomain-runtime-encrypt-secret \
     --from-literal=password=welcome1 -n todo-list

   kubectl -n todo-list label secret tododomain-runtime-encrypt-secret weblogic.domainUID=tododomain
   ```

   Note that the ToDo List example application is pre-configured to use these credentials.
   If you want to use different credentials, you will need to rebuild the Docker images for the example application.
   For the source code of the example application, see the
   [ToDo List Lift-and-Shift Application](https://github.com/verrazzano/examples/tree/master/todo-list) page.  

1. Apply the ToDo List example resources to deploy the application.
   ```
   kubectl apply -f .
   ```

1. Wait for the ToDo List example application to be ready.
   You may need to repeat this command several times before it is successful.
   The `tododomain-adminserver` pod can take some time to be created and `Ready`.
   ```
   kubectl wait pod --for=condition=Ready tododomain-adminserver -n todo-list
   ```

1. Get the `EXTERNAL_IP` address of the `istio-ingressgateway` service.
   ```
   kubectl get service istio-ingressgateway -n istio-system

   NAME                   TYPE           CLUSTER-IP    EXTERNAL-IP   PORT(S)                      AGE
   istio-ingressgateway   LoadBalancer   10.96.97.98   11.22.33.44   80:31380/TCP,443:31390/TCP   13d
   ```   

1. Access the ToDo List example application.
   By default, the application is deployed with a host value of `todo.example.com`.
   There are several ways to access it:
   * **Using the command line**
     ```
     curl -H "Host: todo.example.com" http://11.22.33.44/todo/
     ```
   * **Local testing with a browser** \
     Temporarily, modify the `/etc/hosts` file (on Mac or Linux)
     or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10),
     to add an entry mapping `todo.example.com` to the ingress gateway's `EXTERNAL-IP` address.
     For example:
     ```
     11.22.33.44 todo.example.com
     ```
     Then, you can access the application in a browser at `http://todo.example.com/todo`.
   * **Using your own DNS name:**
     * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address.
     * In this case, you would need to have edited the `todo-list-application.yaml` file
       to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`),
       before deploying the ToDo List application.
     * Then, you can use a browser to access the application at `http://<yourhost.your.domain>/todo/`.

   Accessing the application in a browser will open a page titled "Derek's ToDo List"
   with an edit field and an **Add** button that allows you to add tasks.

1. A variety of endpoints are available to further explore the logs, metrics, and such, associated with
   the deployed ToDo List application.
   Accessing them may require the following:

   * Run this command to get the password that was generated for the telemetry components:
     ```
     kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo
     ```
     The associated user name is `verrazzano`.

   * You will have to accept the certificates associated with the endpoints.

   You can retrieve the list of available ingresses with following command:

   ```
   kubectl get ingress -n verrazzano-system
   NAME                         CLASS    HOSTS                                                     ADDRESS           PORTS     AGE
   verrazzano-console-ingress   <none>   verrazzano.default.140.141.142.143.xip.io                 140.141.142.143   80, 443   7d2h
   vmi-system-api               <none>   api.vmi.system.default.140.141.142.143.xip.io             140.141.142.143   80, 443   7d2h
   vmi-system-es-ingest         <none>   elasticsearch.vmi.system.default.140.141.142.143.xip.io   140.141.142.143   80, 443   7d2h
   vmi-system-grafana           <none>   grafana.vmi.system.default.140.141.142.143.xip.io         140.141.142.143   80, 443   7d2h
   vmi-system-kibana            <none>   kibana.vmi.system.default.140.141.142.143.xip.io          140.141.142.143   80, 443   7d2h
   vmi-system-prometheus        <none>   prometheus.vmi.system.default.140.141.142.143.xip.io      140.141.142.143   80, 443   7d2h
   vmi-system-prometheus-gw     <none>   prometheus-gw.vmi.system.default.140.141.142.143.xip.io   140.141.142.143   80, 443   7d2h
   ```

   Using the ingress host information, some of the endpoints available are:

   | Description | Address | Credentials |
   | ----------- | ------- | ----------- |
   | Kibana      | `https://[vmi-system-kibana ingress host]`     | `verrazzano`/`telemetry-password` |
   | Grafana     | `https://[vmi-system-grafana ingress host]`    | `verrazzano`/`telemetry-password` |
   | Prometheus  | `https://[vmi-system-prometheus ingress host]` | `verrazzano`/`telemetry-password` |

## Troubleshooting

1. Verify that the application configuration, domain, and ingress trait all exist.
   ```
   kubectl get ApplicationConfiguration -n todo-list
   NAME           AGE
   todo-appconf   19h

   kubectl get Domain -n todo-list
   NAME          AGE
   todo-domain   19h

   kubectl get IngressTrait -n todo-list
   NAME                           AGE
   todo-domain-trait-7cbd798c96   19h
   ```

1. Verify that the WebLogic Administration Server and MySQL pods have been created and are running.
   Note that this will take several minutes.
   ```
   kubectl get pods -n todo-list

   NAME                     READY   STATUS    RESTARTS   AGE
   mysql-5c75c8b7f-vlhck    1/1     Running   0          19h
   tododomain-adminserver   2/2     Running   0          19h
   ```

   Copyright (c) 2020, 2021, Oracle and/or its affiliates.
