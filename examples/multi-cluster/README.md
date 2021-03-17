# Multi-Cluster Examples

This repository contains examples of how to use the Verrazzano multi-cluster feature using different approaches.

| Example | Description |
|-------------|-------------|
| [hello-helidon](hello-helidon/) | Hello World Helidon example deployed to a multi-cluster environment. |

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/multi-cluster`.

## Multi-Cluster Installation

This section is common to the examples.

Install Verrazzano on two separate Kubernetes clusters following the [installation instructions](https://verrazzano.io/docs/setup/install/installation/).  
* Install Verrazzano using the `dev` profile on one cluster, this is known as the `admin` cluster.
* Install Verrazzano using the `managed-cluster` profile on the second cluster, this is known as the `managed cluster`.  The `managed-cluster` profile only contains the installation of components that are required on a managed cluster.

Create the environment variables `KUBECONFIG_ADMIN` and `KUBECONFIG_MANAGED1` to point to the kubeconfig files for the admin and managed clusters respectively.

## Register Managed Cluster

This section is common to the examples.

1. Obtain the Kubernetes server address for the admin cluster.  These steps vary based on the Kubernetes platform.
    ```
    # Kind Cluster
    # List the kubeconfig contexts and get the internal config for the admin cluster
    kubectl config get-contexts
    ADMIN_K8S_SERVER_ADDRESS="$(kind get kubeconfig --internal --name <insert context name of admin cluster> | grep "server:" | awk '{ print $2 }')"
   
    # Kubeconfig with a single context
    ADMIN_K8S_SERVER_ADDRESS="$(KUBECONFIG=$KUBECONFIG_ADMIN kubectl config view -o jsonpath={'.clusters[0].cluster.server'})"
    ```

1. Create a ConfigMap that contains the Kubernetes server address of the admin cluster.
    ```
    KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f <<EOF -
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: verrazzano-admin-cluster
      namespace: verrazzano-mc
    data:
      server: "${ADMIN_K8S_SERVER_ADDRESS}"
    EOF
    ```

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



Copyright (c) 2020, 2021, Oracle and/or its affiliates.
