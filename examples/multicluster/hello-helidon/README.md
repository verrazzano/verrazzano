# Multicluster Hello World Helidon

The Hello World Helidon example is a Helidon-based service that returns a "Hello World" response when invoked. The example application is specified using Open Application Model (OAM) component and application configuration YAML files, and then deployed by applying those files.  This example shows how to deploy the Hello World Helidon application in a multicluster environment.

## Prerequisites

1. Create a [multicluster](../README.md/#multicluster-installation) Verrazzano installation.

2. [Register the managed cluster](../README.md/#register-the-managed-cluster).

The Hello World Helidon application deployment artifacts are contained in the Verrazzano project located at
`<VERRAZZANO_HOME>/examples/multicluster/hello-helidon`, where `<VERRAZZANO_HOME>` is the root of the Verrazzano project.

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/multicluster/hello-helidon`.

## Create the application namespace

Apply the `VerrazzanoProject` resource on the admin cluster that defines the namespace for the application.  The namespaces defined in the `VerrazzanoProject` resource will be created on the admin cluster and all the managed clusters.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f verrazzano-project.yaml
   ```

## Deploy the Hello World Helidon application

1. Apply the `hello-helidon` multicluster resources to deploy the application.  Each multicluster resource is an envelope that contains an OAM resource and a list of clusters to which to deploy.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-hello-helidon-comp.yaml
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-hello-helidon-app.yaml
   ```

1. Wait for the application to be ready on the managed cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl wait --for=condition=Ready pods --all -n hello-helidon --timeout=300s
   ```

## Testing the example application

Follow the [instructions](../../hello-helidon/README.md/#testing-the-example-application) for testing the Hello World Helidon application in a single cluster use case. Use the managed-cluster `kubeconfig` for testing the example application.

## Troubleshooting

Follow the [instructions](../../hello-helidon/README.md/#troubleshooting) for troubleshooting the Hello World Helidon application in a single cluster use case. Use the managed-cluster `kubeconfig` for troubleshooting the example application.

1. Verify that the application namespace exists on the managed cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get namespace hello-helidon
   ```

1. Verify that the multicluster resources for the application all exist.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterComponent -n hello-helidon
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterApplicationConfiguration -n hello-helidon
   ```
## Changing the placement of the application to a different cluster

By default, the application will be placed in the managed cluster called `managed1`. You can change the application
placement to be in a different cluster, which can be the admin cluster or a different managed cluster. In this example,
we will change the placement of the application to be in the admin cluster, by patching the multicluster resources
as follows.

1. Patch the `hello-helidon` multicluster resources to change their placement to be in the admin cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl patch mcappconf hello-helidon-appconf -n hello-helidon --type merge --patch "$(cat patch-change-placement-to-admin.yaml)"
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl patch mccomp hello-helidon-component -n hello-helidon --type merge --patch "$(cat patch-change-placement-to-admin.yaml)"
   ```
1. View the multicluster resources to see that the placement has changed to be in the cluster named `local`, which is
   the admin cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get mccomp hello-helidon-component -n hello-helidon -o jsonpath='{.spec.placement}';echo
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get mcappconf hello-helidon-appconf -n hello-helidon -o jsonpath='{.spec.placement}';echo
   ```
1. Patch the VerrazzanoProject to change its placement to be in the admin cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl patch vp hello-helidon -n verrazzano-mc --type merge --patch "$(cat patch-change-placement-to-admin.yaml)"
   ```
1. Wait for the application to be ready on the admin cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl wait --for=condition=Ready pods --all -n hello-helidon --timeout=300s
   ```

You can now test the example application running in the admin cluster.

## Undeploy the Hello World Helidon application

To undeploy the application, irrespective of whether it is placed in the managed cluster or the admin cluster,
delete the multicluster resources and the project from the admin cluster:

```shell
# Delete the multicluster application configuration
$ KUBECONFIG=$KUBECONFIG_ADMIN kubectl delete -f mc-hello-helidon-app.yaml
# Delete the multicluster components for the application 
$ KUBECONFIG=$KUBECONFIG_ADMIN kubectl delete -f mc-hello-helidon-comp.yaml
# Delete the project
$ KUBECONFIG=$KUBECONFIG_ADMIN kubectl delete -f verrazzano-project.yaml
```

Copyright (c) 2021, Oracle and/or its affiliates.
