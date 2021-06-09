# ToDo List

ToDo List is an example application containing a WebLogic component.
For more information and the source code of this application, see the [Verrazzano Examples](https://github.com/verrazzano/examples).

## Prerequisites

* Set up a multicluster Verrazzano environment following the [installation instructions](https://verrazzano.io/docs/setup/multicluster/multicluster/).
* The example assumes that there is a managed cluster named `managed1` associated with the multicluster environment.
If your environment does not have a cluster of that name you may edit the deployment files and change the cluster name
listed in the `placement` section.
* To download the example application image, you must first accept the license agreement.
  * In a browser, navigate to https://container-registry.oracle.com/ and sign in.
  * Search for `example-todo` and select the image name in the results.
  * Click Continue, then read and accept the license agreement.

The ToDo List application deployment artifacts are contained in the Verrazzano project located at
`<VERRAZZANO_HOME>/examples/todo-list`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.\
**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/todo-list`.

## Deploy the example application

1. Create a namespace for the multicluster ToDo List example by applying the Verrazzano project file.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f verrazzano-project.yaml
   ```

1. Login to the `container-registry.oracle.com` docker registry in which the todo list application image is deployed.  You
will require the updated docker config.json, containing you authentication token, for the next step.
   ```
   $ docker login container-registry.oracle.com 
   ```
1. Update the `mc-docker-registry-secret.yaml` file with the your registry authentication info.  Edit the file and replace the 
`<BASE 64 ENCODED DOCKER CONFIG JSON>` with the value generated from the following command.
   ```
   $ cat ~/.docker/config.json | base64
   ```
1. Create a `docker-registry` secret to enable pulling the ToDo List example image from the registry by applying the
`mc-docker-registry-secret.yaml` file.  The multicluster secret resource will generate the required secret in the `mc-todo-list` 
namespace.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-docker-registry-secret.yaml
   ```
1. Create the secrets for the WebLogic domain by applying the mc-weblogic-domain-secret.yaml and mc-runtime-encrypt-secret.yaml files:
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-weblogic-domain-secret.yaml

   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-runtime-encrypt-secret.yaml
   ```

   Note that the ToDo List example application is preconfigured to use these credentials.
   If you want to use different credentials, you will need to rebuild the Docker images for the example application.
   For the source code of this example application, see the
   [ToDo List Lift-and-Shift Application](https://github.com/verrazzano/examples/tree/master/todo-list) page.  

1. Apply the ToDo List example multicluster application resources to deploy the application.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f todo-list-components.yaml

   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f todo-list-application.yaml
   ```

1. Wait for the ToDo List example application to be ready.
   You may need to repeat this command several times before it is successful.
   The `tododomain-adminserver` pod may take a while to be created and `Ready`.
   ```
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl wait pod --for=condition=Ready tododomain-adminserver -n mc-todo-list
   ```
1. Get the generated host name for the application.
   ```
   $ HOST=$(KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get gateway -n mc-todo-list -o jsonpath={.items[0].spec.servers[0].hosts[0]})
   $ echo $HOST
   todo-appconf.mc-todo-list.11.22.33.44.nip.io
   ```

1. Get the `EXTERNAL_IP` address of the `istio-ingressgateway` service.
   ```
   $ ADDRESS=$(KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get service -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
   $ echo $ADDRESS
   11.22.33.44
   ```   

1. Access the ToDo List example application. There are several ways to access it:

   * **Using the command line**
     ```
     $ curl -sk https://${HOST}/todo/ --resolve ${HOST}:443:${ADDRESS}
     ```
     If you are using `nip.io`, then you do not need to include `--resolve`.
   * **Local testing with a browser**

     Temporarily, modify the `/etc/hosts` file (on Mac or Linux)
     or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10),
     to add an entry mapping the host name to the ingress gateway's `EXTERNAL-IP` address.
     For example:
     ```
     11.22.33.44 todo.example.com
     ```
     Then, you can access the application in a browser at `https://todo.example.com/todo`.
   * **Using your own DNS name**
     * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address.
     * In this case, you would need to have edited the `todo-list-application.yaml` file
       to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`),
       before deploying the ToDo List application.
     * Then, you can use a browser to access the application at `https://<yourhost.your.domain>/todo/`.

   Accessing the application in a browser will open a page titled "Derek's ToDo List"
   with an edit field and an **Add** button that allows you to add tasks.

1. A variety of endpoints are available to further explore the logs, metrics, and such, associated with
   the deployed ToDo List application.
   Accessing them may require the following:
   
   * Run this command to get the password that was generated for the telemetry components:
     ```
     $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo
     ```
     The associated user name is `verrazzano`.

   * You will have to accept the certificates associated with the endpoints.

   You can retrieve the list of available ingresses with following command:

   ```
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get ingress -n verrazzano-system
   NAME                         CLASS    HOSTS                                                     ADDRESS           PORTS     AGE
   verrazzano-ingress           <none>   verrazzano.default.140.141.142.143.nip.io                 140.141.142.143   80, 443   7d2h
   vmi-system-es-ingest         <none>   elasticsearch.vmi.system.default.140.141.142.143.nip.io   140.141.142.143   80, 443   7d2h
   vmi-system-grafana           <none>   grafana.vmi.system.default.140.141.142.143.nip.io         140.141.142.143   80, 443   7d2h
   vmi-system-kibana            <none>   kibana.vmi.system.default.140.141.142.143.nip.io          140.141.142.143   80, 443   7d2h
   vmi-system-prometheus        <none>   prometheus.vmi.system.default.140.141.142.143.nip.io      140.141.142.143   80, 443   7d2h
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
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get ApplicationConfiguration -n mc-todo-list
   NAME           AGE
   todo-appconf   19h

   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get Domain -n mc-todo-list
   NAME          AGE
   todo-domain   19h

   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get IngressTrait -n mc-todo-list
   NAME                           AGE
   todo-domain-trait-7cbd798c96   19h
   ```

1. Verify that the WebLogic Administration Server and MySQL pods have been created and are running.
   Note that this will take several minutes.
   ```
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get pods -n mc-todo-list

   NAME                     READY   STATUS    RESTARTS   AGE
   mysql-5c75c8b7f-vlhck    1/1     Running   0          19h
   tododomain-adminserver   2/2     Running   0          19h
   ```

Copyright (c) 2020, 2021, Oracle and/or its affiliates.
