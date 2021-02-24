# Bob's Books

The Bob's Books example is an application based on WebLogic, Helidon, and Coherence. For more information and the source code of this application, see the [Verrazzano examples](https://github.com/verrazzano/examples).

## Prerequisites

* Install Verrazzano following the [installation instructions](https://verrazzano.io/docs/setup/install/installation/).
* In order to be able to download the example image, you must first accept the license agreement.
  * In a browser navigate to https://container-registry.oracle.com/ and sign in.
  * Search for `example-bobbys-coherence`, `example-bobbys-front-end`, `example-bobs-books-order-manager` and `example-roberts-coherence`.
  * For each, select the image name in the results, click Continue, then read and accept the license agreement.

   The Bob's Books application deployment artifacts are contained in the Verrazzano project located at 
   `<VERRAZZANO_HOME>/examples/bobs-books`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

   **NOTE:** All files and paths in this document are relative to `<VERRAZZANO_HOME>/examples/bobs-books`.

## Deploy the example application

1. Create a namespace for the example and add a label identifying the namespace as managed by Verrazzano.

    ```
    $ kubectl create namespace bobs-books
    $ kubectl label namespace bobs-books verrazzano-managed=true istio-injection=enabled
    ```

1. Create a `docker-registry` secret to enable pulling the example image from the registry.
   ```
   $ kubectl create secret docker-registry bobs-books-repo-credentials \
           --docker-server=container-registry.oracle.com \
           --docker-username=YOUR_REGISTRY_USERNAME \
           --docker-password=YOUR_REGISTRY_PASSWORD \
           --docker-email=YOUR_REGISTRY_EMAIL \
           -n bobs-books
   ```
   
   Replace `YOUR_REGISTRY_USERNAME`, `YOUR_REGISTRY_PASSWORD`, and `YOUR_REGISTRY_EMAIL`
   with the values you use to access the registry.  
      
1. Create and label secrets for the WebLogic domains:
    ```
    $ kubectl create secret generic bobbys-front-end-weblogic-credentials --from-literal=password=<password> --from-literal=username=<username> -n bobs-books

    $ kubectl create secret generic bobbys-front-end-runtime-encrypt-secret --from-literal=password=<password> -n bobs-books
    $ kubectl label secret bobbys-front-end-runtime-encrypt-secret weblogic.domainUID=bobbys-front-end -n bobs-books

    $ kubectl create secret generic bobs-bookstore-weblogic-credentials --from-literal=password=<password> --from-literal=username=<username> -n bobs-books

    $ kubectl create secret generic bobs-bookstore-runtime-encrypt-secret --from-literal=password=<password> -n bobs-books
    $ kubectl label secret bobs-bookstore-runtime-encrypt-secret weblogic.domainUID=bobs-bookstore -n bobs-books

    $ kubectl create secret generic mysql-credentials \
        --from-literal=username=<username> \
        --from-literal=password=<password> \
        --from-literal=url=jdbc:mysql://mysql.bobs-books.svc.cluster.local:3306/books \
        -n bobs-books
    ```
   Note that the example application is pre-configured to use specific credentials.
   For the source code for the example application, see
   [Bob's Books example application page](https://github.com/verrazzano/examples/tree/master/bobs-books).
   If you want to use credentials that are different from what is specified in the source code, you will need to rebuild the Docker images for the example application.

1. Apply the example resources to deploy the application.
   ```
   $ kubectl apply -f .
   ```

1. Wait for all of the pods in the Bob's Books example application to be ready.
   You may need to repeat this command several times before it is successful.
   The WebLogic Server and Coherence pods can take some time to be created and `Ready`.
   ```
   $ kubectl wait --for=condition=Ready pods --all -n bobs-books --timeout=600s
   ```

1. Get the `EXTERNAL_IP` address of the `istio-ingressgateway` service.
    ```
    $ kubectl get service -n "istio-system" "istio-ingressgateway" -o jsonpath={.status.loadBalancer.ingress[0].ip}

    11.22.33.44
    ```

1. Get the generated host name for the application.
   ```
   $ kubectl get gateway bobs-books-bobs-books-gw -n bobs-books -o jsonpath={.spec.servers[0].hosts[0]}
   bobs-books.bobs-books.11.22.33.44.xip.io
   ```

1. Access the application. To access the application in a browser, you will need to do one of the following:
    * **Option 1:** If you are using `xip.io`, then you can just access the application using the generated host name, for example:

      a. Robert's Books UI at `https://bobs-books.bobs-books.11.22.33.44.xip.io/`.

      b. Bobby's Books UI at `https://bobs-books.bobs-books.11.22.33.44.xip.io/bobbys-front-end`.

      c. Bob's order manager  UI at `https://bobs-books.bobs-books.11.22.33.44.xip.io/bobs-bookstore-order-manager/orders`.

    * **Option 2:** Temporarily, modify the `/etc/hosts` file (on Mac or Linux) or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10), to add an entry mapping the host used by the application to the external IP address assigned to your gateway. For example:
      ```
      11.22.33.44 bobs-books.example.com
      ```
      Then, you can use a browser to access the application as shown below:
      
      a. Robert's Books UI at `https://bobs-books.example.com/`.

      b. Bobby's Books UI at `https://bobs-books.example.com/bobbys-front-end`.

      c. Bob's order manager  UI at `https://bobs-books.example.com/bobs-bookstore-order-manager/orders`.

    * **Option 3:** Alternatively, point your own DNS name to the load balancer's external IP address. In this case, you would need to have edited the `bobs-books-app.yaml` file to use the appropriate values under the `hosts` section for the application (such as `your-roberts-books-host.your.domain`), before deploying the application.
      Then, you can use a browser to access the application as shown below:

      a. Robert's Books UI at `http://<your-roberts-books-host.your.domain>/`.

      b. Bobby's Books UI at `http://<your-bobbys-books-host.your.domain>/bobbys-front-end`.

      a. Bob's order manager UI at `http://<your-bobs-orders-host.your.domain>/`.

## Troubleshooting
    
1. Verify that the application configuration, domains, Coherence resources, and ingress trait all exist.
   ```
   $ kubectl get ApplicationConfiguration -n bobs-books
   $ kubectl get Domain -n bobs-books
   $ kubectl get Coherence -n bobs-books
   $ kubectl get IngressTrait -n bobs-books
   ```   

1. Verify that the service pods are successfully created and transition to the ready state.
   Note that this may take a few minutes and that you may see some of the services terminate and restart.
   ```
    kubectl get pods -n bobs-books
   
    NAME                                               READY   STATUS    RESTARTS   AGE
    bobbys-coherence-0                                 1/1     Running   0          14m
    bobbys-front-end-adminserver                       2/2     Running   0          6m38s
    bobbys-front-end-managed-server1                   2/2     Running   0          6m2s
    bobbys-helidon-stock-application-bd864fc6d-wbnx7   2/2     Running   0          14m
    bobs-bookstore-adminserver                         2/2     Running   0          10m
    bobs-bookstore-managed-server1                     2/2     Running   0          8m36s
    mysql-5749897f84-gmcgg                             1/1     Running   0          14m
    robert-helidon-7cd74f7b49-d8nph                    2/2     Running   0          14m
    robert-helidon-7cd74f7b49-zqp6n                    2/2     Running   0          14m
    roberts-coherence-0                                1/1     Running   0          10m
    roberts-coherence-1                                1/1     Running   0          14m
   ``` 

Copyright (c) 2020, 2021, Oracle and/or its affiliates.
