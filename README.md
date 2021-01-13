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
    be deployed to a Verrazzano cluster, and install and uninstall Verrazzano components from the cluster in which the operator is deployed.

  - [Examples](./examples) - manifest files for deploying example applications in a Verrazzano managed Kubernetes cluster.

For instructions on using Verrazzano, see the [Verrazzano documentation](https://verrazzano.io/docs/).

For detailed installation instructions, see the [Verrazzano Installation Guide](https://verrazzano.io/docs/setup/install/installation/).

If you want to build and install Verrazzano from this repository, follow the instructions in the [operator](./operator) directory.  

If you are interested in contributing to this repository, see [CONTRIBUTING.md](./CONTRIBUTING.md).
