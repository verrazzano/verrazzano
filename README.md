[![Go Report Card](https://goreportcard.com/badge/github.com/verrazzano/verrazzano)](https://goreportcard.com/report/github.com/verrazzano/verrazzano)

# Verrazzano
> **NOTE**: This is an early alpha release of Verrazzano. It is suitable for investigation and education usage. It is not suitable for production use.

## Introduction
Verrazzano is an end-to-end Enterprise Container Platform for deploying cloud-native and traditional applications in multi-cloud and hybrid environments. It is made up of a curated set of open source components – many that you may already use and trust, and some that were written specifically to pull together all of the pieces to make this a cohesive and easy to use platform.

Verrazzano Enterprise Container Platform includes the following capabilities:

- Hybrid and multi-cluster workload management
- Special handling for WebLogic, Coherence, and Helidon applications
- Multi-cluster infrastructure management
- Integrated and pre-wired application monitoring
- Integrated security
- DevOps and GitOps enablement

This repository contains a Kubernetes operator for installing Verrazzano and example applications for use with Verrazzano.

> **NOTE**: This is an early alpha release of Verrazzano. Some features are still in development.

## Install Verrazzano
To install Verrazzano, follow these steps:  
1. Create an [Oracle Cloud Infrastructure Container Engine for Kubernetes (OKE)](https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengoverview.htm) cluster.
2. Launch an [OCI Cloud Shell](https://docs.cloud.oracle.com/en-us/iaas/Content/API/Concepts/cloudshellgettingstarted.htm).
3. Set up a [kubeconfig](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) file in the OCI Cloud Shell for the OKE cluster. See these detailed [instructions](https://docs.cloud.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengdownloadkubeconfigfile.htm).
4. Clone this [repo](https://github.com/verrazzano/verrazzano) into the home directory of the OCI Cloud Shell.
   - `git clone https://github.com/verrazzano/verrazzano`
   - `cd verrazzano`
5. Run the following commands in the OCI Cloud Shell:
   - `export KUBECONFIG=~/.kube/config`
   - `kubectl apply -f deploy/operator.yaml`
   - `kubectl wait --timeout=20m --for=condition=InstallComplete verrazzano/my-verrazzano`
7. (Optional) Run the following command in the OCI Cloud Shell to monitor the installation log:
    - `kubectl logs -f $(kubectl get pod -l job-name=verrazzano-install-my-verrazzano -o jsonpath="{.items[0].metadata.name}")`
6. (Optional) Install some example applications - see [Deploy the example applications](#deploy-the-example-applications) for details.


> **NOTE**: This alpha release of Verrazzano is intended for installation in a single OKE or Oracle Linux Cloud Native Environment (OLCNE) cluster. You should only install Verrazzano in a cluster that can be safely deleted when your evaluation is complete.

## Deploy the example applications

To deploy the example applications, please see the following instructions:

* [Helidon Hello World](./examples/hello-helidon/README.md)
* [Bob's Books](./examples/bobs-books/README.md)
* [Helidon Sock Shop](./examples/sock-shop/README.md)
* [ToDo List](https://github.com/verrazzano/examples/blob/master/todo-list/README.md)



## Verrazzano Helm Chart

The `install/chart` directory contains a Helm chart for Verrazzano that packages together the core elements that will be installed into the Verrazzano Management Cluster - micro operators,
verrazzano-operator, verrazzano-monitoring-operator, and such - into a single Helm chart.

### Chart Parameters

See the `./install/chart/values.yaml` file for the full list of configurable parameters that can be set using
`--set parameter=value` when installing the Helm chart.


## More Information

For more detailed [installation instructions](./install/README.md), see the `install` directory.
