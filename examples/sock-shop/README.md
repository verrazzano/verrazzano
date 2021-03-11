# Helidon Sock Shop

This example application provides a [Helidon](https://helidon.io) implementation of the [Sock Shop Microservices Demo Application](https://microservices-demo.github.io/).
It uses OAM resources to define the application deployment.

## Prerequisites

Install Verrazzano following the [installation instructions](https://verrazzano.io/docs/setup/install/installation/).

The Sock Shop application deployment artifacts are contained in the Verrazzano project located at
`<VERRAZZANO_HOME>/examples/sockshop`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/sockshop`.

## Deploy the Sock Shop application

1. Create a namespace for the Sock Shop application and add a label identifying the namespace as managed by Verrazzano.
   ```
   $ kubectl create namespace sockshop
   $ kubectl label namespace sockshop verrazzano-managed=true
   ```

1. Apply the Sock Shop OAM resources to deploy the application.
   ```
   $ kubectl apply -f sock-shop-comp.yaml
   $ kubectl apply -f sock-shop-app.yaml
   ```

1. Wait for the Sock Shop application to be ready.
   ```
   $ kubectl wait --for=condition=Ready pods --all -n sockshop --timeout=300s
   ```

## Explore the Sock Shop application

The Sock Shop microservices application implements REST API endpoints including:

- `/catalogue` - Returns the Sock Shop catalog.
This endpoint accepts the `GET` HTTP request method.
- `/register` - POST `{
  "username":"xxx",
  "password":"***",
  "email":"foo@example.com",
  "firstName":"foo",
  "lastName":"coo"
}` to create a user. This
endpoint accepts the `POST` HTTP request method.

**NOTE**:  The following instructions assume that you are using a Kubernetes
environment, such as OKE.  Other environments or deployments may require alternative mechanisms for retrieving addresses,
ports, and such.

Follow these steps to test the endpoints:

1. Get the generated host name for the application.
   ```
   $ HOST=$(kubectl get gateway -n sockshop -o jsonpath={.items[0].spec.servers[0].hosts[0]})
   $ echo $HOST
   sockshop-appconf.sockshop.11.22.33.44.xip.io
   ```

1. Get the `EXTERNAL_IP` address of the `istio-ingressgateway` service.
   ```
   $ ADDRESS=$(kubectl get service -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
   $ echo $ADDRESS
   11.22.33.44
   ```   

1. Access the Sock Shop example application.

   There are several ways to access it:
   * **Using the command line**:

     ```
     # Get catalogue
     $ curl -sk -X GET https://${HOST}/catalogue --resolve ${HOST}:443:${ADDRESS}
     [{"count":115,"description":"For all those leg lovers out there....", ...}]

     # Add a new user (replace values of username and password)
     $ curl -i --header "Content-Type: application/json" --request POST --data '{"username":"foo","password":"****","email":"foo@example.com","firstName":"foo","lastName":"foo"}' -k https://${HOST}/register --resolve ${HOST}:443:${ADDRESS}

     # Add an item to the user's cart
     $ curl -i --header "Content-Type: application/json" --request POST --data '{"itemId": "a0a4f044-b040-410d-8ead-4de0446aec7e","unitPrice": "7.99"}' -k https://${HOST}/carts/{username}/items --resolve ${HOST}:443:${ADDRESS}

     # Get cart items
     $ curl -i -k https://${HOST}/carts/{username}/items --resolve ${HOST}:443:${ADDRESS}
     ```
     If you are using `xip.io`, then you do not need to include `--resolve`.

   * **Local testing with a browser**:

     Temporarily, modify the `/etc/hosts` file (on Mac or Linux)
     or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10),
     to add an entry mapping the host name to the ingress gateway's `EXTERNAL-IP` address.
     For example:
     ```
     11.22.33.44 sockshop.example.com
     ```
     Then, you can access the application in a browser at `https://sockshop.example.com/catalogue`.

   * **Using your own DNS name**:

     * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address.
     * In this case, you would need to edit the `sock-shop-app.yaml` file
       to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`),
       before deploying the Sock Shop application.
     * Then, you can use a browser to access the application at `https://<yourhost.your.domain>/catalogue`.

## Troubleshooting

1. Verify that the application configuration, domain, and ingress trait all exist.
   ```
   $ kubectl get ApplicationConfiguration -n sockshop
   $ kubectl get Domain -n sockshop
   $ kubectl get IngressTrait -n sockshop
   ```   

1. Verify that the Sock Shop service pods are successfully created and transition to the `READY` state. Note that this may take a few minutes and that you may see some of the services terminate and restart.
   ```
    $ kubectl get pods -n sockshop

    NAME             READY   STATUS        RESTARTS   AGE
    carts-coh-0      1/1     Running       0          41s
    catalog-coh-0    1/1     Running       0          40s
    orders-coh-0     1/1     Running       0          39s
    payment-coh-0    1/1     Running       0          37s
    shipping-coh-0   1/1     Running       0          36s
    users-coh-0      1/1     Running       0          35s
   ```
1. A variety of endpoints are available to further explore the logs, metrics, and such, associated with
the deployed Sock Shop application.  Accessing them may require the following:

    - Run this command to get the password that was generated for the telemetry components:
        ```
        $ kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo
        ```
        The associated user name is `verrazzano`.

    - You will have to accept the certificates associated with the endpoints.

    You can retrieve the list of available ingresses with following command:

    ```
    $ kubectl get ing -n verrazzano-system
    NAME                         CLASS    HOSTS                                                    ADDRESS          PORTS     AGE
    verrazzano-console-ingress   <none>   verrazzano.default.140.238.94.217.xip.io                 140.238.94.217   80, 443   7d2h
    vmi-system-api               <none>   api.vmi.system.default.140.238.94.217.xip.io             140.238.94.217   80, 443   7d2h
    vmi-system-es-ingest         <none>   elasticsearch.vmi.system.default.140.238.94.217.xip.io   140.238.94.217   80, 443   7d2h
    vmi-system-grafana           <none>   grafana.vmi.system.default.140.238.94.217.xip.io         140.238.94.217   80, 443   7d2h
    vmi-system-kibana            <none>   kibana.vmi.system.default.140.238.94.217.xip.io          140.238.94.217   80, 443   7d2h
    vmi-system-prometheus        <none>   prometheus.vmi.system.default.140.238.94.217.xip.io      140.238.94.217   80, 443   7d2h
    vmi-system-prometheus-gw     <none>   prometheus-gw.vmi.system.default.140.238.94.217.xip.io   140.238.94.217   80, 443   7d2h
    ```  

    Using the ingress host information, some of the endpoints available are:

    | Description| Address | Credentials |
    | --- | --- | --- |
    | Kibana | `https://[vmi-system-kibana ingress host]` | `verrazzano`/`telemetry-password` |
    | Grafana | `https://[vmi-system-grafana ingress host]` | `verrazzano`/`telemetry-password` |
    | Prometheus | `https://[vmi-system-prometheus ingress host]` | `verrazzano`/`telemetry-password` |    

Copyright (c) 2020, 2021, Oracle and/or its affiliates.
