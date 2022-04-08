## Verrazzano Coherence Workload

This directory contains sample Open Application Model (OAM) resources for testing the VerrazzanoCoherenceWorkload.

## Deploy the Product Catalog Service from Coherence Helidon Sockshop sample to Verrazzano

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

As a basic validation, perform an HTTP GET against /catalogue/size endpoint

    $ curl -sk https://${HOST}/catalogue/size --resolve ${HOST}:443:${ADDRESS}

which should return JSON response: {"size":9}

If you are using nip.io, then you do not need to include --resolve.

Undeploy the application
To undeploy the application, delete the OAM resources for the sample

    $ kubectl delete -f <application directory>/hello-coherence-app.yaml -n hello-coherence
    $ kubectl delete -f <application directory>/hello-coherence-comp.yaml -n hello-coherence

Delete the namespace hello-coherence after the application pods are terminated.

    $ kubectl delete namespace hello-coherence

Copyright (c) 2022, Oracle and/or its affiliates.
