# Verrazzano

Verrazzano is an end-to-end enterprise container platform for deploying cloud native and traditional applications in multicloud and hybrid environments. It is made up of a curated set of open source components – many that you may already use and trust, and some that were written specifically to pull together all of the pieces that make Verrazzano a cohesive and easy to use platform.

Verrazzano includes the following capabilities:

* Hybrid and multicluster workload management
* Special handling for WebLogic, Coherence, and Helidon applications
* Multicluster infrastructure management
* Integrated and pre-wired application monitoring
* Integrated security
* DevOps and GitOps enablement


Select [Quick Start](https://verrazzano.io/latest/docs/quickstart/) to get started.

Verrazzano [release versions](https://github.com/verrazzano/verrazzano/releases/) and source code are available at [https://github.com/verrazzano/verrazzano](https://github.com/verrazzano/verrazzano).
This repository contains a Kubernetes operator for installing Verrazzano and example applications for use with Verrazzano.

For documentation from all releases, see the [Documentation Archive](https://verrazzano.io/archive/docs/).

## Distribution layout

The Verrazzano distribution includes the following artifacts:

* `verrazzano-<major>.<minor>.<patch>/`
  * `README.md`
  * `LICENSE`: The Universal Permissive License (UPL).
  * `bin/`    
     * `linux-amd64/vz`: Verrazzano command-line interface for Linux AMD64 architecture.
     * `linux-arm64/vz`: Verrazzano command-line interface for Linux ARM64 architecture.
     * `darwin-amd64/vz`: Verrazzano command-line interface for Darwin AMD64 architecture.
     * `darwin-arm64/vz`: Verrazzano command-line interface for Darwin ARM64 architecture.
     * `vz-registry-image-helper.sh, bom_utils.sh`:  Helper scripts to download the images from the bill of materials (BOM).
  * `manifests/`     
     * `k8s/verrazzano-platform-operator.yaml`: Kubernetes manifests to deploy the Verrazzano platform operator.
     * `charts/verrazzano-platform-operator/`: Verrazzano Platform Operator Helm chart.
     * `verrazzano-bom.json`: Bill of materials (BOM) containing the list of container images for Verrazzano and the components installed by Verrazzano.
    * `profiles/`
      * `dev.yaml`: The standard `dev` profile to install Verrazzano.
      * `prod.yaml`: The standard `prod` profile to install Verrazzano.
      * `managed-cluster.yaml`: The standard `managed-cluster` profile to install Verrazzano 
  * `images/`:  Verrazzano Enterprise Container Platform archives for private registry install.

## Install Verrazzano 

Install Verrazzano using the instructions in the [Verrazzano Installation Guide](https://verrazzano.io/latest/docs/setup/install/installation/).

## Support

*    For troubleshooting information, see [Diagnostic Tools](https://verrazzano.io/latest/docs/troubleshooting/diagnostictools/) in the Verrazzano documentation.
*    For instructions on using Verrazzano, see the [Verrazzano documentation](https://verrazzano.io/latest/docs/).
*    If you have any questions about Verrazzano, contact us through our [Slack channel](https://bit.ly/3gOeRJn).
*    Use [My Oracle Support](https://support.oracle.com/) to request support.
