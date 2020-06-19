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
3. Clone this repo in the Cloud Shell home.
4. Run the following scripts:  
   - `export CLUSTER_TYPE=OKE`
   - `export VERRAZZANO_KUBECONFIG=<path to valid kubernetes config>`
   - `kubectl create secret docker-registry ocr --docker-username=<username> --docker-password=<password> --docker-server=container-registry.oracle.com`
   - `./install/1-install-istio.sh`
   - `./install/2a-install-system-components-magicdns.sh`
   - `./install/3-install-verrazzano.sh`
   - `./install/4-install-keycloak.sh`

> **NOTE**: This alpha release of Verrazzano is intended for installation in a single OKE or Kind cluster. You should only install Verazzano in a cluster that can be safely deleted when your evaluation is complete.

## More Information
More detailed installation instructions can be found in the `install` folder in this repository.

Example applications can be found in the `examples` folder.

For additional information, see the [Verrazzano documentation](https://verrazzano.io/doc).
