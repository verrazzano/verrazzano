# ToDo List

ToDo List is a demo application which contains a WebLogic component and an IngressTrait. For more information and the code of this application, see the [ToDo List example application page](https://github.com/verrazzano/examples/tree/master/todo-list).

## Deploy the example application
1. Install verrazzano-application-operator by following the [installation instructions](../../install/README.md).
(TODO:  Note that it is assumed that Istio is already installed through the Verrazzano installation.  Will we install Istio here?)

1. Create a Docker image for the ToDo list example app from source code. Detailed instructions on how to do this can be found in the main [ToDo List example application page](https://github.com/verrazzano/examples/tree/master/todo-list).

1. Create a namespace for the ToDo example.  Mark it as a domain namespace for the WebLogic operator that was deployed in the verrazzano-application-operator install above.
   ```
   kubectl create namespace todo
   kubectl label namespace todo verrazzano-domain=true
   ```

1. Create a `docker-registry` secret to enable pulling the ToDo example image.

   ```
   kubectl create secret docker-registry ocir \
           --docker-server=phx.ocir.io \
           --docker-username=YOUR_USERNAME \
           --docker-password=YOUR_PASSWORD \
           --docker-email=YOUR_EMAIL
           -n todo
   ```

   Replace `YOUR_USERNAME`, `YOUR_PASSWORD` and `YOUR_EMAIL` with the values that you
   use to access the registry.  
   (TODO: change this when the example image is moved from ocir )

1. Create and label secrets for the WebLogic domains:

   ```
    kubectl create secret generic tododomain-weblogic-credentials \
      --from-literal=password=welcome1 \
      --from-literal=username=weblogic -n todo


    kubectl create secret generic tododomain-jdbc-tododb \
      --from-literal=password=welcome1 \
      --from-literal=username=derek -n todo
    kubectl -n todo label secret tododomain-jdbc-tododb weblogic.domainUID=tododomain


    kubectl create secret generic tododomain-runtime-encrypt-secret --from-literal=password=welcome1 -n todo
    kubectl -n todo label secret tododomain-runtime-encrypt-secret weblogic.domainUID=tododomain
   ```

   Note that the example application is pre-configured to use these credentials.
   If you want to use different credentials, you will need to rebuild the
   Docker images for the example application.  The source code for the example
   applications is ...  
   (TODO: move ToDo example image and source and update the above)

1. Edit the component configuration file `todo-comp.yaml`, and replace the pattern `%TODO_APP_IMAGE%` with the Docker image you built for the ToDo List application.

1. Apply the component and application configuration for the ToDo example.
   ```
   kubectl apply -f examples/todo
   ```
1. Verify that the application configuration, domain and ingress trait all exist.
   ```
   kubectl get ApplicationConfiguration -n todo
   kubectl get Domain -n todo
   kubectl get IngressTrait -n todo
   ```   
1. Verify that the WebLogic admin server is created by the weblogic-operator.  Note that this will take several minutes and involve the running of an introspector job pod by the operator.
   ```
    kubectl get pods -n todo -w
    NAME                            READY   STATUS            RESTARTS   AGE
    tododomain-introspector-d82gf   1/1     Running           0          49s
    tododomain-introspector-d82gf   0/1     Completed         0          93s
    tododomain-introspector-d82gf   0/1     Terminating       0          94s
    tododomain-adminserver          0/2     Pending           0          1s
    tododomain-adminserver          0/2     Init:0/1          0          1s
    tododomain-adminserver          0/2     PodInitializing   0          5s
    tododomain-adminserver          0/2     Running           0          7s
    tododomain-adminserver          1/2     Running           0          11s
    tododomain-adminserver          2/2     Running           0          52s   
   ```      
1. Verify that the verrazzano-application-operator has reconciled the IngressTrait and created the Istio Gateway and VirtualService.
   ```
   kubectl get gateway -n todo
   NAME                     AGE
   todo-ingress-rule-0-gw   75m

   kubectl get virtualservice -n todo
   NAME                     GATEWAYS                   HOSTS   AGE
   todo-ingress-rule-0-vs   [todo-ingress-rule-0-gw]   [*]     75m
   ```   

1. Get the external IP of the istio-ingressgateway service.
   ```
   kubectl get service istio-ingressgateway -n istio-system
   NAME                   TYPE           CLUSTER-IP     EXTERNAL-IP       PORT(S)                      AGE
   istio-ingressgateway   LoadBalancer   10.96.97.98   11.22.33.44   80:31380/TCP,443:31390/TCP   13d
   ```   

1. Access the ToDo example application.
   The application is deployed by default with a host value of `todo.example.com`.  There are several ways to access it:
   * **Using the Command Line:** `curl -v -H "Host: todo.example.com" http://11.22.33.44/todo/`
   * **Local Testing with a Browser:** Temporarily modify the `/etc/hosts` file (on Mac or Linux) or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10), to add an entry mapping `todo.example.com` to the the ingress gateway's external IP address. For example:
       ```
       11.22.33.44 todo.example.com
       ```
       Then, you can access the application in a browser at `http://todo.example.com/todo`
   * **Using your own DNS Name:**
     * Point your own DNS name to the ingress gateway's external IP address
     * In this case, you would need to have edited the `todo-app.yaml` file to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`), before deploying the ToDo application.
     * Then, you can use a browser to access the application at `http://<yourhost.your.domain>/todo/`

   Accessing the application in a browser should take you to a page titled 'Derek's ToDo List' with an edit field and an 'add' button that allows you to add things to a todo list.

   TODO: Note that JDBC data source is not setup and so the example is only partially functional.
