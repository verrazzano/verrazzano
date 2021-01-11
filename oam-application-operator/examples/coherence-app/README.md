# Coherence app

Coherence app is a sample application from the Coherence Operator examples.

The app contains a Coherence component and an IngressTrait. For more information and the code of this application, see https://github.com/oracle/coherence-operator/tree/master/examples/deployment#example-3---adding-a-user-application-role

TODO: This assumes that you have built the Coherence example app image separately, for now we have a pre-built version in OCIR. Move Coherence example image out of OCIR and put source somewhere. Update this to reflect the location.)

## Deploy the example application
1. Install verrazzano-application-operator by following the [installation instructions](../../install/README.md).  

1. Create a namespace for the Coherence example.
   ```
   kubectl create namespace coherence-example
   ```

1. Create a `docker-registry` secret to enable pulling the coherence example image.  

   ```
   kubectl create secret docker-registry ocir \
           --docker-server=phx.ocir.io \
           --docker-username=YOUR_USERNAME \
           --docker-password=YOUR_PASSWORD \
           --docker-email=YOUR_EMAIL
           -n coherence-example
   ```

   Replace `YOUR_USERNAME`, `YOUR_PASSWORD` and `YOUR_EMAIL` with the values that you
   use to access the registry.  
   (TODO: change this when the example image is moved from ocir )

1. Apply the component and application configuration for the Coherence example.
   ```
   kubectl apply -f examples/coherence-app
   ```
1. Verify that the application configuration, coherence and ingress trait all exist.
   ```
   kubectl get ApplicationConfiguration -n coherence-example
   NAME        AGE
   sampleapp   6m7s

   kubectl get Coherence -n coherence-example
   NAME                      CLUSTER           ROLE                      REPLICAS   READY   PHASE
   example-cluster-proxy     example-cluster   example-cluster-proxy     1          1       Ready
   example-cluster-rest      example-cluster   example-cluster-rest      1          1       Ready
   example-cluster-storage   example-cluster   example-cluster-storage   3          3       Ready

   kubectl get IngressTrait -n coherence-example
   NAME                AGE
   sampleapp-ingress   7m28s
   ```   
1. Verify that the Coherence cluster is created by the coherence-operator.  Note that this will take several minutes and involve the running of an introspector job pod by the operator.
   ```
   kubectl get pods -n coherence-example -w
   NAME                        READY   STATUS    RESTARTS   AGE
   example-cluster-proxy-0     1/1     Running   0          7m59s
   example-cluster-rest-0      1/1     Running   0          7m58s
   example-cluster-storage-0   1/1     Running   0          8m
   example-cluster-storage-1   1/1     Running   0          8m
   example-cluster-storage-2   1/1     Running   0          8m
   ```      

1. Verify that the verrazzano-application-operator has reconciled the IngressTrait and created the Istio Gateway and VirtualService.
   ```
   kubectl get gateway -n coherence-example
   NAME                          AGE
   sampleapp-ingress-rule-0-gw   10m

   kubectl get virtualservice -n coherence-example
   NAME                          GATEWAYS                        HOSTS   AGE
   sampleapp-ingress-rule-0-vs   [sampleapp-ingress-rule-0-gw]   [*]     10m
   ```   

1. Get the external IP of the istio-ingressgateway service.
   ```
   kubectl get service istio-ingressgateway -n istio-system
   NAME                   TYPE           CLUSTER-IP    EXTERNAL-IP      PORT(S)                      AGE
   istio-ingressgateway   LoadBalancer   10.11.12.13   193.194.195.196  80:31380/TCP,443:31390/TCP   16h
   ```   

1.  Access the custom `/query` endpoint

    Use the various `CohQL` commands via the `/query` endpoint to access, and mutate data in the Coherence cluster.

    ```
    curl -i -w '\n' -X PUT http://193.194.195.196/query -d '{"query":"create cache foo"}'

    HTTP/1.1 200 OK
    Via: 1.1 10.12.13.14 (McAfee Web Gateway 7.7.2.16.0.26564)
    date: Wed, 16 Dec 2020 12:37:42 GMT
    server: istio-envoy
    Connection: Keep-Alive
    Transfer-Encoding: chunked
    x-envoy-upstream-service-time: 272

    curl -i -w '\n' -X PUT http://193.194.195.196/query -d '{"query":"insert into foo key(\"foo\") value(\"bar\")"}'

    HTTP/1.1 200 OK
    Via: 1.1 10.12.13.14 (McAfee Web Gateway 7.7.2.16.0.26564)
    date: Wed, 16 Dec 2020 12:39:17 GMT
    server: istio-envoy
    Connection: Keep-Alive
    Transfer-Encoding: chunked
    x-envoy-upstream-service-time: 33

    curl -i -w '\n' -X PUT http://193.194.195.196/query -d '{"query":"select key(),value() from foo"}'

    HTTP/1.1 200 OK
    Via: 1.1 10.12.13.14 (McAfee Web Gateway 7.7.2.16.0.26564)
    date: Wed, 16 Dec 2020 12:39:55 GMT
    server: istio-envoy
    Connection: Keep-Alive
    content-type: application/json
    content-length: 29
    x-envoy-upstream-service-time: 28

    {"result":"{foo=[foo, bar]}"}

    curl -i -w '\n' -X PUT http://193.194.195.196/query -d '{"query":"create cache test"}'

    HTTP/1.1 200 OK
    Via: 1.1 10.12.13.14 (McAfee Web Gateway 7.7.2.16.0.26564)
    date: Wed, 16 Dec 2020 12:40:46 GMT
    server: istio-envoy
    Connection: Keep-Alive
    Transfer-Encoding: chunked
    x-envoy-upstream-service-time: 41

    curl -i -w '\n' -X PUT http://193.194.195.196/query -d '{"query":"select count() from test"}'

    HTTP/1.1 200 OK
    Via: 1.1 10.12.13.14 (McAfee Web Gateway 7.7.2.16.0.26564)
    date: Wed, 16 Dec 2020 12:41:20 GMT
    server: istio-envoy
    Connection: Keep-Alive
    content-type: application/json
    content-length: 14
    x-envoy-upstream-service-time: 10

    {"result":"0"}
    ```

1. To delete the Coherence example.
   ```
   kubectl delete -f examples/coherence-app
   ```
