
# Helidon Sock Shop

This example application provides a [Helidon](https://helidon.io) implementation of the [Sock Shop Microservices Demo Application](https://microservices-demo.github.io/).


## Deploy the example application

1. Prerequisites: Install Verrazzano following the [installation instructions](../../README.md).
   The Helidon Sock Shop example application model and binding files are located at `<VERRAZZANO_HOME>/examples/sock-shop`, where `VERRAZZANO_HOME` is the root of the
   Verrazzano project.

   **NOTE:** All paths in this document are relative to `<VERRAZZANO_HOME>/examples/sock-shop`.

1. Deploy the Verrazzano Application Model and Verrazzano Application Binding for the example application.

    Using an Oracle Cloud Infrastructure Container Engine for Kubernetes (also known as OKE) cluster, run the following script:

    ```
    ./install-sock-shop.sh
    ```


   The script deploys the Verrazzano Application Model and Verrazzano Application Binding, waits for the pods in the `sockshop` namespace to be
   ready, and calls one of the endpoints provided by the REST service implemented by the example application.

1. Verify that all the objects have started. Objects are started in the `sockshop`, `verrazzano-system`, and `monitoring`
  namespaces. The following code block shows some of the objects to expect. Objects not related to this example application have been removed from the list.

    ```
    kubectl get all -n sockshop
    NAME                             READY   STATUS    RESTARTS   AGE
    pod/carts-7b8f98c7d9-xntp4       3/3     Running   0          3m31s
    pod/catalogue-6544766df7-slgwl   3/3     Running   0          3m31s
    pod/orders-59c644cd67-rp7mb      3/3     Running   0          3m31s
    pod/payment-b49c788d4-zb55n      3/3     Running   0          3m31s
    pod/shipping-5699d7b7b8-fzrb2    3/3     Running   0          3m31s
    pod/user-7d849bbc8d-5xhqz        3/3     Running   0          3m31s

    NAME                TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
    service/carts       ClusterIP   10.96.142.78   <none>        80/TCP    3m32s
    service/catalogue   ClusterIP   10.96.96.83    <none>        80/TCP    3m32s
    service/orders      ClusterIP   10.96.33.151   <none>        80/TCP    3m32s
    service/payment     ClusterIP   10.96.58.12    <none>        80/TCP    3m32s
    service/shipping    ClusterIP   10.96.14.251   <none>        80/TCP    3m32s
    service/user        ClusterIP   10.96.207.43   <none>        80/TCP    3m32s

    NAME                        READY   UP-TO-DATE   AVAILABLE   AGE
    deployment.apps/carts       1/1     1            1           3m32s
    deployment.apps/catalogue   1/1     1            1           3m32s
    deployment.apps/orders      1/1     1            1           3m32s
    deployment.apps/payment     1/1     1            1           3m32s
    deployment.apps/shipping    1/1     1            1           3m32s
    deployment.apps/user        1/1     1            1           3m32s

    NAME                                   DESIRED   CURRENT   READY   AGE
    replicaset.apps/carts-7b8f98c7d9       1         1         1       3m33s
    replicaset.apps/catalogue-6544766df7   1         1         1       3m33s
    replicaset.apps/orders-59c644cd67      1         1         1       3m33s
    replicaset.apps/payment-b49c788d4      1         1         1       3m33s
    replicaset.apps/shipping-5699d7b7b8    1         1         1       3m33s
    replicaset.apps/user-7d849bbc8d        1         1         1       3m33s
    ```
## Explore the example application

The Sock Shop microservices application implements REST API endpoints including the following:

- `/catalogue` - Returns the sockshop catalogue.
This endpoint accepts the `GET` HTTP request method.
- `/register` - POST `{
  "username":"xxx",
  "password":"***",
  "email":"foo@example.com",
  "firstName":"foo",
  "lastName":"coo"
}` to create a user. This
endpoint accepts the `POST` HTTP request method.

Follow these steps to test the endpoints:

1. Get the IP address and port number for calling the REST service.

    To get the EXTERNAL-IP address for the `istio-ingressgateway` service:

    ```
    SERVER=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq -r '.status.loadBalancer.ingress[0].ip')
    PORT=80
    ```

1. Use the IP address and port number to call the following services:

    ```
    # Get catalogue
    curl -s -X GET http://"${SERVER}":"${PORT}"/catalogue
    [{"count":115,"description":"For all those leg lovers out there....", ...}]

    # Add a new user (replace values of username and password)
    curl -i --header "Content-Type: application/json" --request POST --data '{"username":"foo","password":"****","email":"foo@example.com","firstName":"foo","lastName":"foo"}' http://"${SERVER}":"${PORT}"/register

    # Add an item to the user's cart
    curl -i --header "Content-Type: application/json" --request POST --data '{"itemId": "a0a4f044-b040-410d-8ead-4de0446aec7e","unitPrice": "7.99"}' http://"${SERVER}":"${PORT}"/carts/{username}/items

    # Get cart items
    curl -i http://"${SERVER}":"${PORT}"/carts/{username}/items

    ```

## Uninstall the example application

Run the following script to delete the Verrazzano Application Model and Verrazzano Application Binding for the example application:

    ```
    ./uninstall-sock-shop.sh
    verrazzanobinding.verrazzano.io "sock-shop-binding" deleted
    verrazzanomodel.verrazzano.io "sock-shop-model" deleted
    ```
