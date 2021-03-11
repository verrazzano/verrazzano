# Hello World Helidon

The Helidon Config example is a Helidon-based service that returns a "hello world" response when invoked. The example application is specified using Open Application Model (OAM) component and application configuration YAML files, and then deployed by applying those files.

## Prerequisites

Install Verrazzano following the [installation instructions](https://verrazzano.io/docs/setup/install/installation/).

The Hello World Helidon application deployment artifacts are contained in the Verrazzano project located at
`<VERRAZZANO_HOME>/examples/hello-helidon`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/hello-helidon`.

## Deploy the Hello World Helidon application

1. Create a namespace for the example application and add a label identifying the namespace as managed by Verrazzano.
   ```
   $ kubectl create namespace helidon-config
   $ kubectl label namespace helidon-config verrazzano-managed=true istio-injection=enabled
   ```

1. Apply the `helidon-config` OAM resources to deploy the application.
   ```
   $ kubectl apply -f helidon-config-comp.yaml
   $ kubectl apply -f helidon-config-app.yaml
   ```

1. Wait for the application to be ready.
   ```
   $ kubectl wait --for=condition=Ready pods --all -n helidon-config --timeout=300s
   ```

## Testing the example application

The Hello World Helidon microservices application implements a single REST API endpoint `/greet`, which returns a message `{"message":"Hello World!"}` when invoked.

**NOTE**:  The following instructions assume that you are using a Kubernetes
environment such as OKE.  Other environments or deployments may require alternative mechanisms for retrieving addresses,
ports, and such.

Follow these steps to test the endpoints:

1. Get the generated host name for the application.

   ```
   $ HOST=$(kubectl get gateway helidon-config-helidon-config-appconf-gw -n helidon-config -o jsonpath={.spec.servers[0].hosts[0]})
   $ echo $HOST
   helidon-config-appconf.helidon-config.11.22.33.44.xip.io
   ```

1. Get the `EXTERNAL_IP` address of the `istio-ingressgateway` service.
   ```
   $ ADDRESS=$(kubectl get service -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
   $ echo $ADDRESS
   11.22.33.44
   ```   

1. Access the application.

   There are several ways to access it:
   * **Using the command line**:
     ```
     $ curl -sk -X GET https://${HOST}/greet --resolve ${HOST}:443:${ADDRESS}
     {"message":"Hello World!"}
     ```
     If you are using `xip.io`, then you do not need to include `--resolve`.
   * **Local testing with a browser**:

     Temporarily, modify the `/etc/hosts` file (on Mac or Linux)
     or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10),
     to add an entry mapping the host name to the ingress gateway's `EXTERNAL-IP` address.
     For example:
     ```
     11.22.33.44 helidon-config.example.com
     ```
     Then you can access the application in a browser at `https://<host>/greet`.

   * **Using your own DNS name**:
     * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address.
     * In this case, you would need to edit the `helidon-config-app.yaml` file
       to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`),
       before deploying the `helidon-config` application.
     * Then, you can use a browser to access the application at `https://<yourhost.your.domain>/greet`.

## Troubleshooting

1. Verify that the application configuration, domain, and ingress trait all exist.
   ```
   $ kubectl get ApplicationConfiguration -n helidon-config
   $ kubectl get IngressTrait -n helidon-config
   ```   

1. Verify that the `helidon-config` service pods are successfully created and transition to the `READY` state.
   Note that this may take a few minutes and that you may see some of the services terminate and restart.
   ```
    $ kubectl get pods -n helidon-config

    NAME                                      READY   STATUS    RESTARTS   AGE
    helidon-config-workload-676d97c7d4-wkrj2   2/2     Running   0          5m39s
   ```
1. A variety of endpoints are available to further explore the logs, metrics, and such, associated with
the deployed Hello World Helidon application.  Accessing them may require the following:

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
