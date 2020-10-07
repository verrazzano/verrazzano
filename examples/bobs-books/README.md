# Bob's Books

The Bob's Books is a book store demo application which contains WebLogic, Helidon, and Coherence components. For more information and the code of this application, see the [Verrazzano examples](https://github.com/verrazzano/examples).

## Deploy the example application

1. Prerequisites: Install Verrazzano following the [installation instructions](../../install/README.md).
   The Bob's Books example application model and binding files are contained in the Verrazzano project located at `<VERRAZZANO_HOME>/examples/bobs-books`, where `VERRAZZANO_HOME` is the root of the Verrazzano project.

   **NOTE:** All files and paths in this document are relative to `<VERRAZZANO_HOME>/examples/bobs-books`.

1. Create a `docker-registry` secret to enable pulling images from the Oracle Container
   Registry.  This is needed to pull WebLogic and Coherence images.  Note that you
   may have already created this secret when installing Verrazzano.

   ```
   kubectl create secret docker-registry ocr \
           --docker-server=container-registry.oracle.com \
           --docker-username=YOUR_USERNAME \
           --docker-password=YOUR_PASSWORD \
           --docker-email=YOUR_EMAIL
   ```

   Replace `YOUR_USERNAME`, `YOUR_PASSWORD` and `YOUR_EMAIL` with the values that you
   use to access the Oracle Container Registry.

1. If you have not done so already, in a web browser, navigate to the [Oracle Container Registry](https://container-registry.oracle.com):

  - Select **Middleware**, review, and _Sign in_ to accept the licenses for the WebLogic and Coherence images.

  - Select **Verrazzano**, review, and accept the licenses for the four repositories listed at the top of the page: `example-bobbys-coherence`, `example-bobbys-front-end`, `example-bobs-books-order-manager`, and `example-roberts-coherence`.

    You will not be able to pull these images until you have accepted the licenses.

1. Create secrets containing the WebLogic administration credentials for the
   two domains:

   ```
   kubectl create secret generic bobs-bookstore-weblogic-credentials \
           --from-literal=username=weblogic \
           --from-literal=password=welcome1

   kubectl create secret generic bobbys-front-end-weblogic-credentials \
           --from-literal=username=weblogic \
           --from-literal=password=welcome1
   ```

   Note that the example applications are pre-configured to use these credentials.
   If you want to use different credentials, you will need to rebuild the
   Docker images for the example application.  The source code for the example
   applications is in the [examples repository](https://github.com/verrazzano/examples).

1. Create the secret containing the MySQL credentials:

   ```
   kubectl create secret generic mysql-credentials \
           --from-literal=username=books \
           --from-literal=password=WebLogic1234
   ```

1. Deploy the MySQL database:

   ```
   kubectl apply -f mysql.yaml
   ```

1. Deploy the Verrazzano Application Model for Bob's Books:

   ```
   kubectl apply -f bobs-books-model.yaml
   ```

1. Update the Verrazzano Application Binding for Bob's Books with the correct DNS names for
   each of the applications.  This step is optional; you can use the IP
   address to access the applications if you do not have DNS names.

   The provided binding includes the following ingress bindings which use
   `*` as the DNS name.  This will result in Istio Virtual Services that
   do not require a DNS name.  It is recommended that you use real DNS
   names for the applications.  For example, if your zone is called
   `example.com`, you might create DNS A records called `bobbys-books.example.com`,
   `roberts-books.example.com` and `bobs-books.example.com`.

   Update the following section to specify the correct DNS names.  Each
   of the DNS A records should point to the external IP address of the
   `istio-ingressgateway` service in the `istio-system` namespace.

    ```
    ingressBindings:
    - name: "bobbys-ingress"
      dnsName: "*"
    - name: "bobs-ingress"
      dnsName: "*"
    - name: "roberts-ingress"
      dnsName: "*"
    ```

1. Deploy the Verrazzano Application Binding for Bob's Books:

   ```
   kubectl apply -f bobs-books-binding.yaml
   ```

## Access the example application

1. Get the external address of the Istio ingress gateway. Access to the example application is through the ingress gateway of the Istio mesh.

    Run this command to get the external IP address of the Istio ingress gateway:
    ```
    kubectl get service istio-ingressgateway -n istio-system
    ```

    For example, assume the response is:
    ```
    NAME                   TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)                      AGE
    istio-ingressgateway   LoadBalancer   10.96.39.106   123.456.78.901   80:31380/TCP,443:31390/TCP   4d2h
    ```

1. Get the password for the telemetry endpoints.

    Run this command to get the password that was generated for the telemetry components:
    ```
    kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo
    ```
1. Test the example application endpoints.

    The following table shows the application endpoints using the these example values:
    - Environment name: `demo`
    - External IP address: `123.456.78.901`
    - DNS zone: `verrazzano.demo.com`
    - The password specified when creating the secret `bobbys-front-end-weblogic-credentials` is `welcome1`. However, you should use a more secure secret when creating this credential.

    | Description| End Point | Credentials |
    | --- | --- | --- |
    | Bobby's Books | http://123.456.78.901/bobbys-front-end | |
    | Bobby's Books WebLogic Console | http://123.456.78.901/console | `weblogic`/`welcome1` |
    | Robert's Books | http://123.456.78.901 | |
    | Elasticsearch | https://elasticsearch.vmi.bobs-books-binding.demo.verrazzano.demo.com | `verrazzano`/`telemetry-password` |
    | Kibana | https://kibana.vmi.bobs-books-binding.demo.verrazzano.demo.com | `verrazzano`/`telemetry-password` |
    | Grafana | https://grafana.vmi.bobs-books-binding.demo.verrazzano.demo.com | `verrazzano`/`telemetry-password` |
    | Prometheus | https://prometheus.vmi.bobs-books-binding.demo.verrazzano.demo.com | `verrazzano`/`telemetry-password` |
