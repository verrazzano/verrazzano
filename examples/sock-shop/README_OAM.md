# OAM Helidon Sock Shop

This example application provides a [Helidon](https://helidon.io) implementation of the [Sock Shop Microservices Demo Application](https://microservices-demo.github.io/).
It leverages OAM resources to define the application deployment.

## Deploy the Sock Shop application

1. Prerequisites: Install Verrazzano following the [installation instructions](../../README.md).
   The Sock Shop application deployment artifacts are contained in the Verrazzano project located at 
   `<VERRAZZANO_HOME>/examples/sockshop`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

   **NOTE:** All files and paths in this document are relative to 
   `<VERRAZZANO_HOME>/examples/sockshop`.

1. Create a namespace for the Sock Shop application and add a label identifying the namespace as managed by Verrazzano.
   ```
   kubectl create namespace sockshop
   kubectl label namespace sockshop verrazzano-managed=true
   ```

1. Apply the Sock Shop OAM resources to deploy the application.
   ```
   kubectl apply -f sock-shop-comp.yaml
   kubectl apply -f sock-shop-app.yaml
   ```
   
1. Wait for the Sock Shop application to be ready.
   ```
   kubectl wait --for=condition=Ready pods --all -n sockshop --timeout=300s
   ```

## Explore the Sock Shop application

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

1. Get the `EXTERNAL_IP` address of the istio-ingressgateway service.
   ```
   kubectl get service istio-ingressgateway -n istio-system

   NAME                   TYPE           CLUSTER-IP    EXTERNAL-IP   PORT(S)                      AGE
   istio-ingressgateway   LoadBalancer   10.96.97.98   11.22.33.44   80:31380/TCP,443:31390/TCP   13d
   ```   

1. The application is deployed by default with a host value of `sockshop.example.com`.
   
   There are several ways to access it:
   * **Using the command line**

     Use the external IP provided by the previous step to call the following services:

     ```
     # Get catalogue
     curl -s -X GET -H "Host: sockshop.example.com" http://<external IP>/catalogue
     [{"count":115,"description":"For all those leg lovers out there....", ...}]

     # Add a new user (replace values of username and password)
     curl -i --header "Content-Type: application/json" -H "Host: sockshop.example.com" --request POST --data '{"username":"foo","password":"****","email":"foo@example.com","firstName":"foo","lastName":"foo"}' http://<external IP>/register

     # Add an item to the user's cart
     curl -i --header "Content-Type: application/json" -H "Host: sockshop.example.com" --request POST --data '{"itemId": "a0a4f044-b040-410d-8ead-4de0446aec7e","unitPrice": "7.99"}' http://<external IP>/carts/{username}/items

     # Get cart items
     curl -i -H "Host: sockshop.example.com" http://"${SERVER}":"${PORT}"/carts/{username}/items
     ```
   * **Local Testing with a Browser** 
   
     Temporarily modify the `/etc/hosts` file (on Mac or Linux)
     or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10), 
     to add an entry mapping `sockshop.example.com` to the ingress gateway's `EXTERNAL-IP` address.
     For example:
     ```
     11.22.33.44 sockshop.example.com
     ```
     Then, you can access the application in a browser at `http://sockshop.example.com/catalogue`

   * **Using your own DNS Name:**
   
     * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address
     * In this case, you would need to edit the `sock-shop-app.yaml` file 
       to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`), 
       before deploying the Sock Shop application.
     * Then, you can use a browser to access the application at `http://<yourhost.your.domain>/catalogue`
   
1. ## Troubleshooting
    
1. Verify that the application configuration, domain and ingress trait all exist.
   ```
   kubectl get ApplicationConfiguration -n sockshop
   kubectl get Domain -n sockshop
   kubectl get IngressTrait -n sockshop
   ```   

1. Verify that the Sock Shop service pods are successfully created and transition to the ready state.
   Note that this may take a few minutes and that you may see some of the services terminate and restart.
   ```
    kubectl get pods -n sockshop
   
    NAME             READY   STATUS        RESTARTS   AGE
    carts-coh-0      1/1     Running       0          41s
    catalog-coh-0    1/1     Running       0          40s
    orders-coh-0     1/1     Running       0          39s
    payment-coh-0    1/1     Running       0          37s
    shipping-coh-0   1/1     Running       0          36s
    users-coh-0      1/1     Running       0          35s
   ``` 