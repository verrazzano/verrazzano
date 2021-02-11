# Hello World Helidon

The Hello World Helidon example is a Helidon-based service that returns a "hello world" response when invoked. The example application is specified using Open Application Model (OAM) component and application configuration YAML files, and then deployed by applying those files.

## Prerequisites

Install Verrazzano following the [installation instructions](../../README.md).

The Hello World Helidon application deployment artifacts are contained in the Verrazzano project located at
`<VERRAZZANO_HOME>/examples/hello-helidon`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/hello-helidon`.

## Deploy the Hello World Helidon application

1. Create a namespace for the example application and add a label identifying the namespace as managed by Verrazzano.
   ```
   kubectl create namespace oam-hello-helidon
   kubectl label namespace oam-hello-helidon verrazzano-managed=true
   ```

1. Apply the `hello-helidon` OAM resources to deploy the application.
   ```
   kubectl apply -f hello-helidon-comp.yaml
   kubectl apply -f hello-helidon-app.yaml
   ```

1. Wait for the application to be ready.
   ```
   kubectl wait --for=condition=Ready pods --all -n oam-hello-helidon --timeout=300s
   ```

## Testing the example application

The Hello World Helidon microservices application implements a single REST API endpoint `/greet`, which returns a message `{"message":"Hello World!"}` when invoked.

**NOTE**:  This following set of instructions assumes you are using a Kubernetes
environment such as OKE.  Other environments or deployments may require alternate mechanisms for retrieving addresses,
ports and such.

Follow these steps to test the endpoints:

1. Get the `EXTERNAL_IP` address of the istio-ingressgateway service.  

   ```
   kubectl get service istio-ingressgateway -n istio-system

   NAME                   TYPE           CLUSTER-IP    EXTERNAL-IP   PORT(S)                      AGE
   istio-ingressgateway   LoadBalancer   10.96.97.98   11.22.33.44   80:31380/TCP,443:31390/TCP   13d
   ```   

1. The application is deployed by default with a host value of `hello-helidon.example.com`.

   There are several ways to access it:
   * **Using the command line**

     Use the external IP provided by the previous step to call the `/greet` endpoint:

     ```
     curl -s -X GET -H "Host: hello-helidon.example.com" http://<external IP>/greet
     {"message":"Hello World!"}
     ```
   * **Local testing with a browser**

     Temporarily modify the `/etc/hosts` file (on Mac or Linux)
     or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10),
     to add an entry mapping `hello-helidon.example.com` to the ingress gateway's `EXTERNAL-IP` address.
     For example:
     ```
     11.22.33.44 hello-helidon.example.com
     ```
     Then, you can access the application in a browser at `http://hello-helidon.example.com/greet`.

   * **Using your own DNS name:**

     * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address.
     * In this case, you would need to edit the `hello-helidon-app.yaml` file
       to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`),
       before deploying the `hello-helidon` application.
     * Then, you can use a browser to access the application at `http://<yourhost.your.domain>/greet`.

## Troubleshooting

1. Verify that the application configuration, domain, and ingress trait all exist.
   ```
   kubectl get ApplicationConfiguration -n oam-hello-helidon
   kubectl get IngressTrait -n oam-hello-helidon
   ```   

1. Verify that the `hello-helidon` service pods are successfully created and transition to the ready state.
   Note that this may take a few minutes and that you may see some of the services terminate and restart.
   ```
    kubectl get pods -n oam-hello-helidon

    NAME                                      READY   STATUS    RESTARTS   AGE
    hello-helidon-workload-676d97c7d4-wkrj2   2/2     Running   0          5m39s
   ```
1. A variety of endpoints are available to further explore the logs, metrics and such associated with
the deployed Hello World Helidon application.  Accessing them may require the following:

    - Run this command to get the password that was generated for the telemetry components:
        ```
        kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo
        ```
        The associated user name is `verrazzano`

    - You will have to accept the certificates associated with the endpoints.

    You can retrieve the list of available ingresses with following command:

    ```
    kubectl get ing -n verrazzano-system
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
