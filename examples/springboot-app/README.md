# Spring Boot Application

This example provides a simple web application developed using Spring Boot. For more information and the source code of this application, see the [Verrazzano examples](https://github.com/verrazzano/examples).

## Prerequisites

Install Verrazzano following the [installation instructions](https://verrazzano.io/docs/setup/install/installation/).

  The Spring Boot example application deployment artifacts are contained in the Verrazzano project located at `<VERRAZZANO_HOME>/examples/springboot-app`, where `VERRAZZANO_HOME` is the root of the Verrazzano project.

   **NOTE:** All files and paths in this document are relative to `<VERRAZZANO_HOME>/examples/springboot-app`.


## Deploy the example application

1. Create a namespace for the Spring Boot application and add a label identifying the namespace as managed by Verrazzano.
   ```
   $ kubectl create namespace springboot
   $ kubectl label namespace springboot verrazzano-managed=true
   ```

1. Apply the Spring Boot OAM resources to deploy the application.
   ```
   $ kubectl apply -f springboot-comp.yaml
   $ kubectl apply -f springboot-app.yaml
   ```

1. Wait for the Spring Boot application to be ready.
   ```
   $ kubectl wait --for=condition=Ready pods --all -n springboot --timeout=300s
   ```

## Access the example application

1. Get the generated host name for the application.
   ```
   $ HOST=$(kubectl get gateway -n springboot -o jsonpath={.items[0].spec.servers[0].hosts[0]})
   $ echo $HOST
   springboot-appconf.springboot.11.22.33.44.xip.io
   ```

1. Get the `EXTERNAL_IP` address of the `istio-ingressgateway` service.
   ```
   $ ADDRESS=$(kubectl get service -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
   $ echo $ADDRESS
   11.22.33.44
   ```   

1. Access the Springboot example application.
   There are several ways to access it:

   * **Using the command line**
     ```
     $ curl -sk https://${HOST} --resolve ${HOST}:443:${ADDRESS}
     $ curl -sk https://${HOST}/facts --resolve ${HOST}:443:${ADDRESS}
     ```
     If you are using `xip.io` then you do not need to include `--resolve`.
   * **Local testing with a browser** \
     Temporarily, modify the `/etc/hosts` file (on Mac or Linux)
     or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10),
     to add an entry mapping the host name to the ingress gateway's `EXTERNAL-IP` address.
     For example:
     ```
     11.22.33.44 springboot.example.com
     ```
     Then, you can access the application in a browser at `https://springboot.example.com/` and `https://springboot.example.com/facts`.
   * **Using your own DNS name:**
     * Point your own DNS name to the ingress gateway's `EXTERNAL-IP` address.
     * In this case, you would need to have edited the `springboot-app.yaml` file
       to use the appropriate value under the `hosts` section (such as `yourhost.your.domain`),
       before deploying the Spring Boot application.
     * Then, you can use a browser to access the application at `https://<yourhost.your.domain>/` and `https://<yourhost.your.domain>/facts`.

    The actuator endpoint is accessible under the path `/actuator` and the Prometheus endpoint exposing metrics data in a format that can be scraped by a Prometheus server is accessible under the path `/actuator/prometheus`.

1. A variety of endpoints are available to further explore the logs and metrics associated with
   the deployed Spring Boot application.
   Accessing them may require the following:

   * Run this command to get the password that was generated for the telemetry components:
     ```
     $ kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo
     ```
     The associated user name is `verrazzano`.

   * You will have to accept the certificates associated with the endpoints.

   You can retrieve the list of available ingresses with following command:

   ```
   $ kubectl get ingress -n verrazzano-system
   NAME                         CLASS    HOSTS                                                     ADDRESS           PORTS     AGE
   verrazzano-console-ingress   <none>   verrazzano.default.140.141.142.143.xip.io                 140.141.142.143   80, 443   7d2h
   vmi-system-api               <none>   api.vmi.system.default.140.141.142.143.xip.io             140.141.142.143   80, 443   7d2h
   vmi-system-es-ingest         <none>   elasticsearch.vmi.system.default.140.141.142.143.xip.io   140.141.142.143   80, 443   7d2h
   vmi-system-grafana           <none>   grafana.vmi.system.default.140.141.142.143.xip.io         140.141.142.143   80, 443   7d2h
   vmi-system-kibana            <none>   kibana.vmi.system.default.140.141.142.143.xip.io          140.141.142.143   80, 443   7d2h
   vmi-system-prometheus        <none>   prometheus.vmi.system.default.140.141.142.143.xip.io      140.141.142.143   80, 443   7d2h
   vmi-system-prometheus-gw     <none>   prometheus-gw.vmi.system.default.140.141.142.143.xip.io   140.141.142.143   80, 443   7d2h
   ```

   Using the ingress host information, some of the endpoints available are:

   | Description | Address | Credentials |
   | ----------- | ------- | ----------- |
   | Kibana      | `https://[vmi-system-kibana ingress host]`     | `verrazzano`/`telemetry-password` |
   | Grafana     | `https://[vmi-system-grafana ingress host]`    | `verrazzano`/`telemetry-password` |
   | Prometheus  | `https://[vmi-system-prometheus ingress host]` | `verrazzano`/`telemetry-password` |


## Undeploy the example application   

1. Delete the Spring Boot OAM resources to undeploy the application.
   ```
   $ kubectl delete -f springboot-app.yaml
   $ kubectl delete -f springboot-comp.yaml
   ```

1. Delete the namespace `springboot` after the application pod is terminated.
   ```
   $ kubectl get pods -n springboot
   $ kubectl delete namespace springboot
   ```

Copyright (c) 2020, 2021, Oracle and/or its affiliates.
