# Verrazzano Commercial Edition

Verrazzano Commercial Edition is a fully-featured enterprise container platform for deploying cloud native and traditional applications in multicloud
and hybrid environments, packaged for the enterprise.

# Overview
Verrazzano Commercial Edition includes the following capabilities:

- Hybrid and multicluster workload management
- Special handling for WebLogic, Coherence, and Helidon applications
- Multicluster infrastructure management
- Integrated and pre-wired application monitoring
- Integrated security
- DevOps and GitOps enablement

# Components
Verrazzano Commercial Edition includes a curated set of open source components – many that you may already use and trust, 
and some that were written specifically to pull together all the pieces to make this a cohesive and easy to use platform.

| Component                    | Version | Description                                                                              |
|------------------------------|---------|------------------------------------------------------------------------------------------|
| alert-manager                | 0.24.0  | Handles alerts sent by client applications, such as the Prometheus server.               |
| cert-manager                 | 1.7.1   | Automates the management and issuance of TLS certificates.                               |
| Coherence Operator           | 3.2.6   | Assists with deploying and managing Coherence clusters.                                  |
| ExternalDNS                  | 0.10.2  | Synchronizes exposed Kubernetes Services and ingresses with DNS providers.               |
| Fluentd                      | 1.14.5  | Collects logs and sends them to OpenSearch.                                              |
| Grafana                      | 7.5.15  | Tool to help you examine, analyze, and monitor metrics.                                  |
| Istio                        | 1.13.5  | Service mesh that layers transparently onto existing distributed applications.           |
| Jaeger                       | 1.34.1  | Distributed tracing system for monitoring and troubleshooting distributed systems.       |
| Keycloak                     | 15.0.2  | Provides single sign-on with Identity and Access Management.                             |
| Kiali                        | 1.42.0  | Management console for the Istio service mesh.                                           |
| kube-state-metrics           | 2.4.2   | Provides metrics about the state of Kubernetes API objects.                              |
| MySQL                        | 8.0.29  | Open source relational database management system used by Keycloak.                      |
| NGINX Ingress Controller     | 1.1.1   | Traffic management solution for cloud‑native applications in Kubernetes.                 |
| Node Exporter                | 1.3.1   | Prometheus exporter for hardware and OS metrics.                                         |
| OAM Kubernetes Runtime       | 0.3.0   | Plug-in for implementing the Open Application Model (OAM) control plane with Kubernetes. |
| OpenSearch                   | 1.2.3   | Provides a distributed, multitenant-capable full-text search engine.                     |
| OpenSearch Dashboards        | 1.2.0   | Provides search and data visualization capabilities for data indexed in OpenSearch.      |
| Prometheus                   | 2.34.0  | Provides event monitoring and alerting.                                                  |
| Prometheus Adapter           | 0.9.1   | Provides metrics in support of pod autoscaling.                                          |
| Prometheus Operator          | 0.55.1  | Provides management for Prometheus monitoring tools.                                     |
| Prometheus Pushgateway       | 1.4.2   | Allows ephemeral and batch jobs to expose their metrics to Prometheus.                   |
| Rancher                      | 2.6.7   | Manages multiple Kubernetes clusters.                                                    |
| WebLogic Kubernetes Operator | 3.4.0   | Assists with deploying and managing WebLogic domains.                                    |

## Distribution layout

The Verrazzano Commercial Edition distribution includes the following artifacts:

* `verrazzano-<major>.<minor>.<patch>/`
  * `README.md`
  * `README.html` 
  * `LICENSE`: The Universal Permissive License (UPL).
  * `bin/`    
     * `linux-amd64/vz`: Verrazzano command-line interface for Linux AMD64 architecture.
     * `linux-arm64/vz`: Verrazzano command-line interface for Linux ARM64 architecture.
     * `darwin-amd64/vz`: Verrazzano command-line interface for Darwin AMD64 architecture.
     * `darwin-arm64/vz`: Verrazzano command-line interface for Darwin ARM64 architecture.
     * `vz-registry-image-helper.sh, bom_utils.sh`:  Helper scripts to download the images from the bill of materials (BOM).
  * `manifests/`     
     * `k8s/verrazzano-platform-operator.yaml`: Kubernetes manifests to deploy the Verrazzano platform operator.
     * `charts/verrazzano-platform-operator/`: Helm chart for the Verrazzano Platform Operator.
     * `verrazzano-bom.json`: Bill of materials (BOM) containing the list of Docker images for Verrazzano and the components installed by Verrazzano.
  * `images/`:  Verrazzano Enterprise Container Platform archives for private registry install.

## Install Verrazzano using a private container registry

You can install Verrazzano using a private Docker-compliant container registry. This requires the following:

*    Load the product images into your private registry. This includes loading all the required Verrazzano container images from `/images` into your own registry and repository.
*    Install Verrazzano using the instructions in the [Verrazzano Installation Guide](https://verrazzano.io/latest/docs/setup/install/installation/).

## Support

*    If you have any questions about Verrazzano Commercial Edition, contact us through our [Slack channel](https://bit.ly/3gOeRJn).
*    For instructions about using Verrazzano, see the [Verrazzano documentation](https://verrazzano.io/latest/docs/).
*    For detailed installation instructions, see the [Install Guide](https://verrazzano.io/latest/docs/setup/install/installation/) in the Verrazzano documentation.
*    For troubleshooting information, see [Diagnostic Tools](https://verrazzano.io/latest/docs/troubleshooting/diagnostictools/) in the Verrazzano documentation.
*    To report a bug or request for an enhancement, submit them through [bugs or enhancements requests](https://github.com/verrazzano/verrazzano/issues/new/choose) in GitHub.
