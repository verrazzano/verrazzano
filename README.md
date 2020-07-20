# Verrazzano Enterprise Container Platform
> **NOTE**: This is an early alpha release of Verrazzano. It is suitable for investigation and education usage. It is not suitable for production use. 

## Introduction
Verrazzano Enterprise Container Platform is a curated collection of open source and Oracle-authored components that form a complete platform for modernizing existing applications, and for deploying and managing your container applications across multiple Kubernetes clusters. 

Verrazzano Enterprise Container Platform includes the following capabilities:

- Hybrid and multi-cluster workload management
- Special handling for WebLogic, Coherence, and Helidon applications
- Multi-cluster infrastructure management
- Integrated and pre-wired application monitoring
- Integrated security
- DevOps and GitOps enablement

This repository contains installation scripts and example applications for use with Verrazzano.

> **NOTE**: This is an early alpha release of Verrazzano. Some features are still in development. 

## tl;dr
To install Verrazzano, follow these steps:  
1. Create an OKE cluster  
2. Start OCI Cloud Shell  
3. Copy the kubeconfig to Cloud Shell
4. Clone this repo in the Cloud Shell home.
   - `git clone https://github.com/verrazzano/verrazzano`
   - `cd verrazzano`
5. Run the following scripts:  
   - `export CLUSTER_TYPE=OKE`
   - `export VERRAZZANO_KUBECONFIG=~/.kube/config`
   - `export KUBECONFIG=~/.kube/config`
   - `kubectl create secret docker-registry ocr --docker-username=<username> --docker-password=<password> --docker-server=container-registry.oracle.com`
   - `./install/1-install-istio.sh`
   - `./install/2a-install-system-components-magicdns.sh`
   - `./install/3-install-verrazzano.sh`
   - `./install/4-install-keycloak.sh`
6. (Optional) Install some example applications - see below for details.

> **NOTE**: This alpha release of Verrazzano is intended for installation in a single OKE or Kind cluster. You should only install Verazzano in a cluster that can be safely deleted when your evaluation is complete.

## Deploying the example applications

To deploy the example applications, please see the following instructions:

* [Bob's Books](./examples/bobs-books/README.md)
* [Helidon Hello World](./examples/hello-helidon/README.md)
* TBD

## More Information

For additional information, see the [Verrazzano documentation](https://verrazzano.io/doc).

More detailed [installation instructions](./install/README.md) can be found in the `install` directory.
