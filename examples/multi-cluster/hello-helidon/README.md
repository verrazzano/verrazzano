# Multi-Cluster Hello World Helidon

The Hello World Helidon example is a Helidon-based service that returns a "hello world" response when invoked. The example application is specified using Open Application Model (OAM) component and application configuration YAML files, and then deployed by applying those files.  This example shows how to deploy the Hello World Helidon application in a multi-cluster environment.

## Prerequisites

Install Verrazzano on two separate Kubernetes clusters following the [installation instructions](https://verrazzano.io/docs/setup/install/installation/).  Install a full Verrazzano on one cluster, this is known as the `admin` cluster.  Install Verrazzano using the `managed-cluster` profile on the second cluster, this is known as a `managed cluster`.

The Hello World Helidon application deployment artifacts are contained in the Verrazzano project located at
`<VERRAZZANO_HOME>/examples/multi-cluster/hello-helidon`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

Create the environment variables `KUBECONFIG_ADMIN` and `KUBECONFIG_MANAGED1` to point to the kubeconfig files for the admin and managed clusters respectively.

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/multi-cluster/hello-helidon`.

## Register the Managed Cluster

1. Obtain the credentials for scraping metrics from the managed cluster.  The script will output the credentials into a file named `managed1.yaml` into the current folder.
   ```
   $ export KUBECONFIG=$KUBECONFIG_MANAGED1
   ../../../platform-operator/scripts/create_managed_cluster_secret.sh -n managed1 -o .
   ```

1. Create a secret on the admin cluster that contains the credentials for scraping metrics from the managed cluster.  The file `managed1.yaml` that was created in the previous step is provided as input to this step.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl create secret generic prometheus-managed1 -n verrazzano-mc --from-file=managed1.yaml
   ```

1. Apply the VerrazzanoManagedCluster object on the admin cluster to begin the registration process for a managed cluster named `managed1`.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f vmc-managed1.yaml
   ```

1. Export the yaml file created to register the managed cluster.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get secret verrazzano-cluster-managed1-manifest -n verrazzano-mc -o jsonpath={.data.yaml} | base64 --decode > register.yaml
   ```

1. Apply the registration file on the managed cluster.
   ```
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl apply -f register.yaml
   ```

## Create the Application Namespace

1. Apply the `VerrazzanoProject` resource on the admin cluster that defines the namespace for the application.  The namespaces defined in the `VerrazzanoProject` resource will be created on the admin cluster and all managed clusters.
   ```
   KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f verrazzano-project.yaml
   ```

1. The synchronization of operations to the managed clusters may take about a minute to complete. 
   ```
   <insert command to wait on status of VerrazzanoProject resource>
   ```

## Deploy the Hello World Helidon application

1. Apply the `hello-helidon` multi-cluster resources to deploy the application.  Each of the multi-cluster resources is an envelope that contains the OAM resource to and list of clusters to deploy to.
   ```
   KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-hello-helidon-comp.yaml
   KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-hello-helidon-app.yaml
   ```

1. Wait for the application to be ready.
   ```
   $ kubectl wait --for=condition=Ready pods --all -n hello-helidon --timeout=300s
   ```

## Testing the example application

The Hello World Helidon microservices application implements a single REST API endpoint `/greet`, which returns a message `{"message":"Hello World!"}` when invoked.

**NOTE**:  The following instructions assume that you are using a Kubernetes
environment such as OKE.  Other environments or deployments may require alternative mechanisms for retrieving addresses,
ports, and such.

Follow these steps to test the endpoints:

1. Get the generated host name for the application.

   ```
   $ HOST=$(kubectl get gateway hello-helidon-hello-helidon-appconf-gw -n hello-helidon -o jsonpath={.spec.servers[0].hosts[0]})
   $ echo $HOST
   hello-helidon-appconf.hello-helidon.11.22.33.44.xip.io
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
      11.22.33.44 hello-helidon.example.com
      ```
      Then you can access the application in a browser at `https://<host>/greet`.

    * **Using your own DNS name**:
        * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address.
        * In this case, you would need to edit the `hello-helidon-app.yaml` file
          to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`),
          before deploying the `hello-helidon` application.
        * Then, you can use a browser to access the application at `https://<yourhost.your.domain>/greet`.

## Troubleshooting

1. Verify that the application configuration, domain, and ingress trait all exist.
   ```
   $ kubectl get ApplicationConfiguration -n hello-helidon
   $ kubectl get IngressTrait -n hello-helidon
   ```   

1. Verify that the `hello-helidon` service pods are successfully created and transition to the `READY` state.
   Note that this may take a few minutes and that you may see some of the services terminate and restart.
   ```
    $ kubectl get pods -n hello-helidon

    NAME                                      READY   STATUS    RESTARTS   AGE
    hello-helidon-workload-676d97c7d4-wkrj2   2/2     Running   0          5m39s
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
