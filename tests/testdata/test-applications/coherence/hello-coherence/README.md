## Coherence Application

Sample Coherence application, for testing the VerrazzanoCoherenceWorkload.

## Requires

[Maven](https://maven.apache.org/download.cgi)

## Steps to build the application
Use maven to build the Spring Boot uber jar.

    $ cd <application root directory>
    $ mvn clean install

## Create a Docker image
The Dockerfile provided in this example uses an Oracle Linux image as the base image, which doesn't include the Java Development Kit (JDK).
The Dockerfile expects `openjdk-<version>_linux-x64_bin.tar.gz` in the project root directory, which is available on the [OpenJDK General-Availability Releases](https://jdk.java.net/archive/) page.
Please check the exact version of the JDK from the Dockerfile and install accordingly.

    $ cd <project root directory>
    $ mvn clean package
    $ docker build -t <container registry>/<image>:<version> .
    $ docker image push <image registry>/<image>:<version>

## Deploy the sample application to Verrazzano

Create a namespace for the sample application and add a label identifying the namespace as managed by Verrazzano. To run this application in the default namespace, skip creating the namespace and do not specify the namespace in all the kubectl commands below.

    $ kubectl create namespace hello-coherence
    $ kubectl label namespace hello-coherence verrazzano-managed=true

Deploy the application, by applying the sample resource

    $ Set the registry URL for the sample application in hello-coherence-comp.yaml
    $ kubectl apply -f <application directory>/hello-coherence-comp.yaml -n hello-coherence
    $ kubectl apply -f <application directory>/hello-coherence-app.yaml -n hello-coherence

Wait for the sample application to be ready.

    $ kubectl get pods -n hello-coherence
    $ or using
    $ kubectl wait --for=condition=Ready pods --all -n hello-coherence --timeout=300s

Get the generated host name for the application.

    $ ADDRESS=$(kubectl get service -n istio-system \
           istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

Get the EXTERNAL_IP address of the istio-ingressgateway service.

    $ HOST=$(kubectl get gateways.networking.istio.io \
           -n hello-coherence \
           -o jsonpath='{.items[0].spec.servers[0].hosts[0]}')

Access the application

    $ curl -X POST -k -H "Content-Type: application/json" -d '{"Sample message"}' https://${HOST}/hello/postMessage --resolve ${HOST}:443:${ADDRESS}
    $ curl -sk https://${HOST}/hello/coherence --resolve ${HOST}:443:${ADDRESS}

If you are using nip.io, then you do not need to include --resolve.


Undeploy the application
To undeploy the application, delete the OAM resources for the sample

    $ kubectl delete -f <application directory>/hello-coherence-app.yaml -n hello-coherence
    $ kubectl delete -f <application directory>/hello-coherence-comp.yaml -n hello-coherence

Delete the namespace hello-coherence after the application pods are terminated.

    $ kubectl delete namespace hello-coherence

Copyright (c) 2022, Oracle and/or its affiliates.
