# Change placement of an application

This example shows how to change the placement of an application from one cluster to another.

## Prerequisites

Follow the [instructions](../hello-helidon/README.md) to deploy the multicluster Hello World Helidon example.  This example will show how to change the placement of the application from the managed cluster `managed` to the admin cluster.

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/multicluster/change-placement`.

## Edit the placement of the application

Edit the placement of each of the application resources by changing the cluster name under `spec.placement.clusters` from `managed1` to `local` (the default name of the admin cluster).  

    ```
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl edit MultiClusterComponent hello-helidon-component -n hello-helidon
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl edit MultiClusterApplicationConfiguration hello-helidon-appconf -n hello-helidon
    ```
## Verify the change in placement

Wait a few minutes for the change in placement to take effect on each cluster.

1. Use the following commands to verify the resources have been removed from the managed cluster.

    ```
    $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterComponent hello-helidon-component -n hello-helidon
    $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl get MultiClusterApplicationConfiguration hello-helidon-appconf -n hello-helidon
    ```

Copyright (c) 2021, Oracle and/or its affiliates.
