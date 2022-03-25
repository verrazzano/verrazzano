## WebLogic Application

Sample WebLogic application, for testing the VerrazzanoWebLogicWorkload.

## Requires

[Maven](https://maven.apache.org/download.cgi)

## Steps to build the application
The bash script setup/build.sh creates the auxiliary image for model in image deployment, by including the sample application under wlsdeploy/applications.

    $ cd <application root directory>
    $ mvn clean package
    $ cd setup; ./build.sh <container registry>/<image>:<version>
    $ docker image push <image registry>/<image>:<version>

## Deploy the sample application to Verrazzano

Create a namespace for the sample application and add a label identifying the namespace as managed by Verrazzano. To run this application in the default namespace, skip creating the namespace and do not specify the namespace in all the kubectl commands below.

    $ kubectl create namespace hello-wls
    $ kubectl label namespace hello-wls verrazzano-managed=true istio-injection=enabled

Create a docker-registry secret to enable pulling the example image from the registry.

    $ kubectl create secret docker-registry hellodomain-repo-credentials \
              --docker-server=container-registry.oracle.com \
              --docker-username=YOUR_REGISTRY_USERNAME \
              --docker-password=YOUR_REGISTRY_PASSWORD \
              --docker-email=YOUR_REGISTRY_EMAIL \
              -n hello-wls
Replace YOUR_REGISTRY_USERNAME, YOUR_REGISTRY_PASSWORD, and YOUR_REGISTRY_EMAIL with the values you use to access the registry.

Create the secrets for the WebLogic domain:

    $ Replace the values of the WLS_USERNAME and WLS_PASSWORD environment variables as appropriate.
    $ export WLS_USERNAME=<username>
    $ export WLS_PASSWORD=<password, must be at least 8 alphanumeric characters with at least one number or special character>
    $ kubectl create secret generic hellodomain-weblogic-credentials --from-literal=password=$WLS_PASSWORD --from-literal=username=$WLS_USERNAME -n hello-wls


Deploy the application, by applying the sample resource

    $ Set the registry URL for the sample application in hello-wls-comp.yaml
    $ kubectl apply -f <application directory>/hello-wls-comp.yaml -n hello-wls
    $ kubectl apply -f <application directory>/hello-wls-app.yaml -n hello-wls

Wait for the sample application to be ready.

    $ kubectl get pods -n hello-wls
    $ or using
    $ kubectl wait pod --for=condition=Ready hellodomain-adminserver -n hello-wls

Get the generated host name for the application.

    $ ADDRESS=$(kubectl get service -n istio-system \
           istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

Get the EXTERNAL_IP address of the istio-ingressgateway service.

    $ HOST=$(kubectl get gateways.networking.istio.io \
           -n hello-wls \
           -o jsonpath='{.items[0].spec.servers[0].hosts[0]}')

Access the application

    $ curl -sk https://${HOST}/hello/weblogic/greetings/message --resolve ${HOST}:443:${ADDRESS}

If you are using nip.io, then you do not need to include --resolve.


Undeploy the application
To undeploy the application, delete the OAM resources for the sample

    $ kubectl delete -f <application directory>/hello-wls-app.yaml -n hello-wls
    $ kubectl delete -f <application directory>/hello-wls-comp.yaml -n hello-wls

Delete the namespace hello-wls after the application pods are terminated. The secrets created for the WebLogic domain also will be deleted.

    $ kubectl delete namespace hello-wls

Copyright (c) 2022, Oracle and/or its affiliates.
