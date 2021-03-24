# Change placement of an application

This example shows how to change the placement of an application from one cluster to another.

## Prerequisites

The multicluster Hello World Helidon example will used to show how to change the placement of an application to another cluster.  Follow the [instructions](../hello-helidon/README.md) to deploy the multicluster Hello World Helidon example.  

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/multicluster/change-placement`.

## Edit the placement of the application

Edit the placement of each of the application resources by changing the cluster name under `spec.placement.clusters` from `managed1` to `local` (the default name of the admin cluster).  Use the following commands to apply the changes.

    ```
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-hello-helidon-comp.yaml
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f mc-hello-helidon-app.yaml
    ```

## Verify the change in placement

Wait a few minutes for the change in placement to take effect on each cluster.

1. Use the following commands to verify the resources have been removed from the managed cluster.

    ```
    $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterComponent hello-helidon-component -n hello-helidon
    $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get mccomp -n hello-helidon
    $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterApplicationConfiguration hello-helidon-appconf -n hello-helidon
    $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get VerrazzanoHelidonWorkload -n hello-helidon
    ```

1. Use the following commands to verify the resources have been moved to the admin cluster.

    ```
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get MultiClusterComponent hello-helidon-component -n hello-helidon
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get mccomp -n hello-helidon
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get MultiClusterApplicationConfiguration hello-helidon-appconf -n hello-helidon
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get VerrazzanoHelidonWorkload -n hello-helidon
    ```

## Testing the example application

Follow the [instructions](../../hello-helidon/README.md/#testing-the-example-application) for testing the Hello World Helidon application in a single cluster use case. Use the admin cluster `kubeconfig` for testing the example application.

## Troubleshooting

Follow the [instructions](../../hello-helidon/README.md/#troubleshooting) for troubleshooting the Hello World Helidon application in a single cluster use case. Use the admin cluster `kubeconfig` for troubleshooting the example application.

