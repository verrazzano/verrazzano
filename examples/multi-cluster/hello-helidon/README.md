# Multi-Cluster Hello World Helidon

The Hello World Helidon example is a Helidon-based service that returns a "hello world" response when invoked. The example application is specified using Open Application Model (OAM) component and application configuration YAML files, and then deployed by applying those files.  This example shows how to deploy the Hello World Helidon application in a multi-cluster environment.

## Prerequisites

Follow the [multi-cluster installation instructions](../README.md/#multi-cluster-installation) to configure an admin cluster and one managed cluster.

Follow the [register managed cluster](../README.md/#register-managed-cluster) to register the managed cluster.

The Hello World Helidon application deployment artifacts are contained in the Verrazzano project located at
`<VERRAZZANO_HOME>/examples/multi-cluster/hello-helidon`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/multi-cluster/hello-helidon`.

## Create the Application Namespace

1. Apply the `VerrazzanoProject` resource on the admin cluster that defines the namespace for the application.  The namespaces defined in the `VerrazzanoProject` resource will be created on the admin cluster and all managed clusters.
   ```
   KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f verrazzano-project.yaml
   ```

## Deploy the Hello World Helidon application

1. Apply the `hello-helidon` multi-cluster resources to deploy the application.  Each of the multi-cluster resources is an envelope that contains the OAM resource to and list of clusters to deploy to.
   ```
   KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-hello-helidon-comp.yaml
   KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-hello-helidon-app.yaml
   ```

1. Wait for the application to be ready on the managed cluster.
   ```
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl wait --for=condition=Ready pods --all -n hello-helidon --timeout=300s
   ```

## Testing the example application

Following the [instructions](../../hello-helidon/README.md/#testing-the-example-application) for testing the Hello Helidon application in a single cluster case. Use the managed-cluster `kubeconfig` for testing the example application.

## Troubleshooting

Following the [instructions](../../hello-helidon/README.md/#troubleshooting) for troubleshooting the Hello Helidon application in a single cluster case. Use the managed-cluster `kubeconfig` for troubleshooting the example application.

1. Verify that the application namespace exists on the managed cluster.
   ```
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get namespace hello-helidon
   ```

1. Verify that the multi-cluster resources for the application all exist.
   ```
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterComponent -n hello-helidon
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterApplicationConfiguration -n hello-helidon
   ```

Copyright (c) 2021, Oracle and/or its affiliates.
