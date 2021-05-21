# Multicluster Hello World Helidon

The Hello World Helidon example is a Helidon-based service that returns a "Hello World" response when invoked. The example application is specified using Open Application Model (OAM) component and application configuration YAML files, and then deployed by applying those files.  This example shows how to deploy the Hello World Helidon application in a multicluster environment.

## Prerequisites

Create a multicluster Verrazzano installation with one admin and one managed cluster, and register the managed cluster, following the instructions [here](https://verrazzano.io/docs/setup/multicluster/multicluster/).

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

Follow the [instructions](../../hello-helidon/README.md/#testing-the-example-application) for testing the Hello World Helidon application in a single cluster use case. Use the managed cluster `kubeconfig` for testing the example application.

## Troubleshooting

Follow the [instructions](../../hello-helidon/README.md/#troubleshooting) for troubleshooting the Hello World Helidon application in a single cluster use case. Use the managed cluster `kubeconfig` for troubleshooting the example application.

1. Verify that the application namespace exists on the managed cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get namespace hello-helidon
   ```

1. Verify that the multicluster resources for the application all exist.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterComponent -n hello-helidon
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterApplicationConfiguration -n hello-helidon
   ```
## Locating the application on a different cluster

By default, the application is located on the managed cluster called `managed1`. You can change the application's location to be on a different cluster, which can be the admin cluster or a different managed cluster. In this example, you change the placement of the application to the admin cluster by patching the multicluster resources.

1. To change the application's location to the admin cluster, specify the change placement patch file.

   ```shell
   # To change the placement to the admin cluster
   $ export CHANGE_PLACEMENT_PATCH_FILE="patch-change-placement-to-admin.yaml"
   ```
   This environment variable is used in subsequent steps.

1. To change their placement, patch the `hello-helidon` multicluster resources.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl patch mcappconf hello-helidon-appconf -n hello-helidon --type merge --patch "$(cat $CHANGE_PLACEMENT_PATCH_FILE)"
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl patch mccomp hello-helidon-component -n hello-helidon --type merge --patch "$(cat $CHANGE_PLACEMENT_PATCH_FILE)"
   ```
1. To verify that their placement has changed, view the multicluster resources.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get mccomp hello-helidon-component -n hello-helidon -o jsonpath='{.spec.placement}';echo
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get mcappconf hello-helidon-appconf -n hello-helidon -o jsonpath='{.spec.placement}';echo
   ```
   The cluster
      name, `local`, indicates placement in the admin cluster.

1. To change its placement, patch the VerrazzanoProject.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl patch vp hello-helidon -n verrazzano-mc --type merge --patch "$(cat $CHANGE_PLACEMENT_PATCH_FILE)"
   ```
1. Wait for the application to be ready on the admin cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl wait --for=condition=Ready pods --all -n hello-helidon --timeout=300s
   ```
   **Note:** If you are returning the application to the managed cluster, then instead, wait for the application to be
   ready on the managed cluster.
   ```shell
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl wait --for=condition=Ready pods --all -n hello-helidon --timeout=300s
   ```

1. Now, you can test the example application running in its new location.

To return the application to the managed cluster named `managed1`, set the value of the `CHANGE_PLACEMENT_PATCH_FILE` environment variable to the patch file provided for that purpose, then repeat the previous numbered steps.

```shell
   # To change the placement back to the managed cluster named managed1
   $ export CHANGE_PLACEMENT_PATCH_FILE="patch-return-placement-to-managed1.yaml"
```

## Undeploy the Hello World Helidon application

Regardless of its location, to undeploy the application,
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
