# ToDo List

ToDo List is an example application containing a WebLogic component.
For more information and the source code of this application, see
[ToDo List example application page](https://github.com/verrazzano/examples/tree/master/todo-list).

## Deploy the example application

1. Prerequisites: Install Verrazzano following the [installation instructions](../../README.md).
   The ToDo List application deployment artifacts are contained in the Verrazzano project located at 
   `<VERRAZZANO_HOME>/examples/todo-list`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

   **NOTE:** All files and paths in this document are relative to 
   `<VERRAZZANO_HOME>/examples/todo-list`.

1. Create a namespace for the ToDo List example and add a label identifying the namespace as managed by Verrazzano.
   ```
   kubectl create namespace todo
   kubectl label namespace todo verrazzano-managed=true
   ```

1. Create a `docker-registry` secret to enable pulling the ToDo List example image from the registry.
   ```
   kubectl create secret docker-registry tododomain-repo-credentials \
           --docker-server=container-registry.oracle.com \
           --docker-username=YOUR_REGISTRY_USERNAME \
           --docker-password=YOUR_REGISTRY_PASSWORD \
           --docker-email=YOUR_REGISTRY_EMAIL
           -n todo
   ```
   
   Replace `YOUR_REGISTRY_USERNAME`, `YOUR_REGISTRY_PASSWORD` and `YOUR_REGISTRY_EMAIL` 
   with the values you use to access the registry.  

1. Create and label secrets for the WebLogic domain:
   ```
   kubectl create secret generic tododomain-weblogic-credentials \
     --from-literal=password=welcome1 \
     --from-literal=username=weblogic -n todo

   kubectl create secret generic tododomain-jdbc-tododb \
     --from-literal=password=welcome1 \
     --from-literal=username=derek -n todo
   
   kubectl -n todo label secret tododomain-jdbc-tododb weblogic.domainUID=tododomain

   kubectl create secret generic tododomain-runtime-encrypt-secret \
     --from-literal=password=welcome1 -n todo
   
   kubectl -n todo label secret tododomain-runtime-encrypt-secret weblogic.domainUID=tododomain
   ```

   Note that the ToDo List example application is pre-configured to use these credentials.
   If you want to use different credentials, you will need to rebuild the Docker images for the example application.
   For the source code for the example applications, see
   [ToDo List example application page](https://github.com/verrazzano/examples/tree/master/todo-list).  

1. Apply the ToDo List example resources to deploy the application.
   ```
   kubectl apply -f .
   ```

1. Wait for the ToDo List example application to be ready.
   ```
   kubectl wait pod --for=condition=Ready tododomain-adminserver -n todo
   ```

1. Get the `EXTERNAL_IP` address of the istio-ingressgateway service.
   ```
   kubectl get service istio-ingressgateway -n istio-system

   NAME                   TYPE           CLUSTER-IP    EXTERNAL-IP   PORT(S)                      AGE
   istio-ingressgateway   LoadBalancer   10.96.97.98   11.22.33.44   80:31380/TCP,443:31390/TCP   13d
   ```   

1. Access the ToDo List example application.
   The application is deployed by default with a host value of `todo.example.com`.
   There are several ways to access it:
   * **Using the Command Line** 
     ```
     curl -H "Host: todo.example.com" http://11.22.33.44/todo/
     ```
   * **Local Testing with a Browser** \
     Temporarily modify the `/etc/hosts` file (on Mac or Linux)
     or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10), 
     to add an entry mapping `todo.example.com` to the ingress gateway's `EXTERNAL-IP` address.
     For example:
     ```
     11.22.33.44 todo.example.com
     ```
     Then, you can access the application in a browser at `http://todo.example.com/todo`
   * **Using your own DNS Name:**
     * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address
     * In this case, you would need to have edited the `todo-list-application.yaml` file 
       to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`), 
       before deploying the ToDo List application.
     * Then, you can use a browser to access the application at `http://<yourhost.your.domain>/todo/`

   Accessing the application in a browser should take you to a page titled 'Derek's ToDo List'
   with an edit field and an 'Add' button that allows you to add tasks.

## Troubleshooting
    
1. Verify that the application configuration, domain and ingress trait all exist.
   ```
   kubectl get ApplicationConfiguration -n todo
   kubectl get Domain -n todo
   kubectl get IngressTrait -n todo
   ```   

1. Verify that the WebLogic admin server is created by the weblogic-operator.
   Note that this will take several minutes and involve the running of an introspector job pod by the operator.
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
