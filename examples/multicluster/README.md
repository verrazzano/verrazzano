# Multicluster Examples

This repository contains examples of different approaches for using Verrazzano multiclusters.

| Example | Description |
|-------------|-------------|
| [hello-helidon](hello-helidon/) | Hello World Helidon example deployed to a multicluster environment. |

**NOTE:** All files and paths in this document are relative to
`<VERRAZZANO_HOME>/examples/multicluster`.

### Multicluster installation

Complete the following sections prior to running the multicluster examples.

1. Install Verrazzano on two separate Kubernetes clusters following the [installation instructions](https://verrazzano.io/docs/setup/install/installation/):
   * On one cluster, install Verrazzano using the `dev` profile; this is known as the `admin` cluster.
   * On the other cluster, install Verrazzano using the `managed-cluster` profile;, this is known as the `managed cluster`.  The `managed-cluster` profile contains only the components that are required on a managed cluster.

2. Create the environment variables `KUBECONFIG_ADMIN` and `KUBECONFIG_MANAGED1` and point them to the `kubeconfig` file for the `admin` and `managed cluster`, respectively.

### Register the managed cluster

1. Obtain the Kubernetes server address for the admin cluster.  These steps will vary depending on the Kubernetes platform.
    ```
    # Kind Cluster
    # List the kubeconfig contexts and get the internal config for the admin cluster
    $ kubectl config get-contexts
    $ ADMIN_K8S_SERVER_ADDRESS="$(kind get kubeconfig --internal --name <insert context name of admin cluster> | grep "server:" | awk '{ print $2 }')"
   
    # Kubeconfig with a single context
    $ ADMIN_K8S_SERVER_ADDRESS="$(KUBECONFIG=$KUBECONFIG_ADMIN kubectl config view -o jsonpath={'.clusters[0].cluster.server'})"
    ```

1. Create a ConfigMap that contains the Kubernetes server address of the admin cluster.
    ```
    $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f <<EOF -
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: verrazzano-admin-cluster
      namespace: verrazzano-mc
    data:
      server: "${ADMIN_K8S_SERVER_ADDRESS}"
    EOF
    ```

1. Obtain the credentials for scraping metrics from the managed cluster.  The script will output the credentials to a file named `managed1.yaml` in the current folder.
   ```
   $ export KUBECONFIG=$KUBECONFIG_MANAGED1
   $ ../../../platform-operator/scripts/create_managed_cluster_secret.sh -n managed1 -o .
   ```

1. Create a secret on the admin cluster that contains the credentials for scraping metrics from the managed cluster.  The file `managed1.yaml` that was created in the previous step provides input to this step.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl create secret generic prometheus-managed1 -n verrazzano-mc --from-file=managed1.yaml
   ```

1. Apply the VerrazzanoManagedCluster object on the admin cluster to begin the registration process for a `managed cluster` named `managed1`.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl apply -f vmc-managed1.yaml
   ```

1. Export the YAML file created to register the managed cluster.
   ```
   $ KUBECONFIG=$KUBECONFIG_ADMIN kubectl get secret verrazzano-cluster-managed1-manifest -n verrazzano-mc -o jsonpath={.data.yaml} | base64 --decode > register.yaml
   ```

1. Apply the registration file on the managed cluster.
   ```
   $ KUBECONFIG=$KUBECONFIG_MANAGED1 kubectl apply -f register.yaml
   ```



Copyright (c) 2021, Oracle and/or its affiliates.
