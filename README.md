# Verrazzano

Verrazzano is an end-to-end enterprise container platform for deploying cloud-native and traditional applications in multi-cloud and hybrid environments. It is made up of a curated set of open source components – many that you may already use and trust, and some that were written specifically to pull together all of the pieces to make this a cohesive and easy to use platform.

Verrazzano includes the following capabilities:

- Hybrid and multi-cluster workload management
- Special handling for WebLogic, Coherence, and Helidon applications
- Multi-cluster infrastructure management
- Integrated and pre-wired application monitoring
- Integrated security
- DevOps and GitOps enablement

This repository contains the following content:

  - [Verrazzano platform operator](./operator) - a [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) that can 
    be deployed to a Verrazzano cluster and install Verrazzano components into and uninstall those components from the cluster in
    which the operator is deployed.

  - [examples](./examples) - manifests for deploying example applications into a Verrazzano managed Kubernetes cluster.


 For more detailed installation instructions, see the [Verrazzano Installation Guide](https://verrazzano.io/docs/setup/install/installation/).

If you are interested in contributing, see [CONTRIBUTING.md](./CONTRIBUTING.md).

## Install

To install the latest Verrazzano release, follow these steps:

1. Deploy the Verrazzano platform operator.

    ```shell
    kubectl apply -f https://github.com/verrazzano/verrazzano/releases/latest/download/operator.yaml
    ```

1. Install Verrazzano with its dev profile.

    ```shell
    kubectl apply -f - <<EOF
    apiVersion: install.verrazzano.io/v1alpha1
    kind: Verrazzano
    metadata:
      name: example-verrazzano
    spec:
      profile: dev
    EOF
    ```

If you want to build and install Verrazzano from this repository, follow the instructions in the [operator](./operator) directory.  For
detailed installation instructions on using Verrazzano, see the [Verrazzano documentation](https://verrazzano.io/docs/).
