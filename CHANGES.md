### v1.4.1
Fixes:

- Updated OpenSearch heap memory settings
- Fixed the WebLogic and Helidon Grafana dashboards so that they display data properly


### v1.4.0
Features:

- Added the Verrazzano command-line tool (CLI) for interactive installation, upgrade, uninstall, cluster analysis, and bug reporting.
- Added backup and restore functionality using Velero and rancher-backup.
- Added Prometheus Operator based metrics collection (using ServiceMonitors and PodMonitors) for both Verrazzano system components and applications.
- Added a new API version for the Verrazzano resource, `install.verrazzano.io/v1beta1`. See the [Deprecated API Migration Guide](https://verrazzano.io/latest/docs/reference/migration").
- Verrazzano distribution `tar.gz` artifacts now include the new CLI binaries and tooling.
- Replaced Elasticsearch and Kibana with OpenSearch and OpenSearch dashboards (pods, URLs, CRD fields).
- Improved Rancher integration.
    - Added the Rancher UI-based Verrazzano console.
    - Keycloak SSO authentication and authorization is configured by default.
    - OCI drivers now are enabled by default and ready-to-use.
- kube-prometheus-stack components now are enabled by default.
- Improved uninstall resiliency and performance.
- Added support for OCNE 1.5.x.
- Added support for Kubernetes v1.24.

Component version updates:

- Coherence Operator v3.2.6
- Istio v1.14.3
- Jaeger v1.34.1
- Rancher v2.6.8

Components added:

- Rancher Backup Operator v2.1.3
- Velero v1.8.1
- Velero Plugin For AWS v1.4.1

Components removed:

- Config Map Reload

Fixes:

- Resolved an issue where Verrazzano started an installation, immediately after an upgrade, but before all the components were ready.
- Resolved an issue where application pods that required an Istio sidecar did not restart after an upgrade.
- Resolved unnecessary temporary file cleanup for Helm overrides after installation or upgrade.
- Resolved an issue with Verrazzano resource status conditions being appended as duplicates instead of updated.
- Resolved an issue where Verrazzano Monitoring Operator was querying OpenSearch before it was ready.
- Resolved an issue where Verrazzano Platform Operator transitioned to a ready condition before all webhook context paths were ready.
- Updated base and other images to resolves CVEs.


### v1.3.5
Component version updates:

- WebLogic Kubernetes Operator v3.4.3

### v1.3.4
Fixes:

- Updated the Kiali image to fix CVEs.
- Resolved an issue with Prometheus volume attachment during upgrade.

Component version updates:

- Rancher v2.6.6

### v1.3.3
Fixes:

- Fixed AuthProxy to emit access logs.
- Fixed Verazzano Console intermittent failures of timing out loading application details.

Component version updates:

- Istio v1.13.5

### v1.3.2
Fixes:

- Fixed Fluentd pattern to correctly parse `severity` value from WebLogic logs.
- Fixed IngressTrait to remove the deleted IngressTrait entries from the Istio Gateway.

### v1.3.1
Fixes:

- Resolved an issue where the Verrazzano uninstall deleted additional namespaces when deleting Rancher components.
- Fixed IngressTrait controller to support Services as component workloads.
- Added liveness probe for the AuthProxy NGINX server.
- Added support for dynamic configuration overrides to Verrazzano components from various monitored sources, including ConfigMaps, Secrets, and Values referenced in the Verrazzano CR.
- Added support for JWT authentication and authorization policy specification for applications.
- Added support for Prometheus Service Monitor and Pod Monitor CRs deployed using Prometheus Operator.
- Updated Keycloak image to fix CVEs.

### v1.3.0
Features:

- Post-installation updates: configurations for DNS, certificate management, logging, ingress, and OpenSearch cluster configuration can be updated after a Verrazzano installation.
- Added Jaeger Distributed Tracing.
- Support for Kubernetes v1.22 and v1.23.
- kube-prometheus-stack components are now part of Verrazzano and can be enabled, these include Prometheus Operator, Alertmanager, kube-state-metrics, and such.

Component version updates:

- cert-manager v1.7.1
- Coherence Operator 3.2.5
- Istio v1.13.2
- Jaeger Operator v1.32.0
- Kiali v1.42.0
- NGINX Ingress Controller v1.1.1
- Node Exporter v1.3.1
- Prometheus v2.34.0
- Rancher v2.6.4
- WebLogic Kubernetes Operator v3.4.0

Components added:

- Alertmanager v0.24.0
- kube-state-metrics v2.4.2
- Prometheus Adapter v0.9.1
- Prometheus Operator v0.55.1
- Prometheus Pushgateway v1.4.2

Fixes:

- Resolved an issue in the console UI with displaying multicluster applications when a managed cluster is partially registered.
- Resolved an issue in the console UI with the display of the Bob's Books sample WebLogic application.
- Resolved an issue with exporting WebLogic application metrics in a private registry installation of Verrazzano.

### v1.2.2
Fixes:

- Resolved an issue with the Grafana Dashboards for Helidon in multicluster setup.
- Resolved an issue with naming the Istio Authorization Policy for the AuthProxy.
- Resolved an issue with AuthProxy pods being evicted due to ephemeral storage.
- Resolved an issue with the length of the cookie TTL in the ingress trait.

### v1.2.1
Fixes:

- Resolved an issue with upgrade when configured to use a private registry.
- Resolved an issue with the public image of WebLogic Monitoring Exporter being used when a private registry is configured.
- Resolved an issue with intermittent upgrade failures while upgrading from Verrazzano v1.0.2 to v1.2.0.
- Resolved an issue with the console UI when viewing WebLogic applications.
- Resolved an issue with the console UI when displaying an application that is targeted to a managed cluster that has not completed the registration process.
- Resolved an issue with the console UI not displaying the traits for an OAM application.
- Resolved an issue with the `verrazzano-application-operator` pod continually crashing and restarting.
- Resolved an issue with the WebLogic workload `logHome` value being ignored and always using `/scratch/log`.
- Resolved an issue with Prometheus not scraping metrics from Verrazzano managed namespaces that do not have Istio injection enabled.
- The Verrazzano operators no longer have watches on resources in the `kube-system` namespace.
- Updated Keycloak image to address CVEs.

Known Issues:

- Importing a Kubernetes v1.21 cluster into Rancher might not work properly. Rancher does not currently support Kubernetes v1.21.

### v1.2.0
Features:

- Logging enhancements:
    - Added support for Oracle Cloud Infrastructure Logging integration.
    - Replaced Elasticsearch and Kibana with Opensearch and Opensearch Dashboard.
    - Updated Opensearch `prod` profile data node configuration to 3 replicas.
    - Enhanced Fluentd parsing/filtering rules for Verrazzano system logs.
- Added support for using `instance_principal` authorization with using Oracle Cloud Infrastructure DNS.
- Added support for metrics integration with non-OAM applications.
- Added support for scaling Istio gateways and setting affinity.
- Added support for scaling Verrazzano AuthProxy and setting affinity.
- Component version updates:
    - External DNS v0.10.2.
    - MySQL v8.0.28.
    - Grafana v7.5.11.
    - Prometheus v2.31.1.
    - Opensearch v1.2.3 (replaces Elasticsearch).
    - Opensearch Dashboards v1.2.0 (replaces Kibana).
    - WebLogic Kubernetes Operator v3.3.7.

Fixes:

- Fixed Keycloak issue creating incorrect `verrazzano-monitors` group on installation.
- Fixed Verrazzano failing to uninstall in a private registry configuration due to a missing Rancher image.
- Fixed Rancher installation when `tls-ca-additional` secret is not present.
- Fixed Opensearch parsing errors of `trait` field.
- Fixed Custom CA certificates support.
- Fixed issue requeuing unsupported traits in the Verrazzano Application Operator, and updated the OAM Operator.
- Aligned Helidon workload service port names with Istio conventions to avoid protocol defaulting to TCP in all cases.
- Added ability to set a DestinationRule with HTTP Cookie for session affinity.

Known Issues:

- Importing a Kubernetes v1.21 cluster into Rancher might not work properly. Rancher does not currently support Kubernetes v1.21.

### v1.1.2
Fixes:
- Fixed installation to create `verrazzano-monitors` group correctly.
- Fixed installation to enable network access to Prometheus for Kiali.
- Updated Spring Boot example image to address CVEs.
- Updated Kibana image to address CVEs.
- Updated Elasticsearch image to address CVEs.
- Fixed Verrazzano failing to install when specifying a custom CA certificate.
- Updated Keycloak image to address CVEs.
- Fixed Verrazzano failing to install when the `spec.components.certManager.certificate.acme.environment` field was set to `production` in the Verrazzano CR.
- Added support for using private DNS and instance principals with Oracle Cloud Infrastructure DNS.
- Fixed Verrazzano failing to uninstall in a private registry configuration due to a missing Rancher image.
- Updated Verrazzano to use the Rancher v2.5.9 Helm chart.

Known Issues:
- Importing a Kubernetes v1.21 cluster into Rancher might not work properly. Rancher does not currently support Kubernetes v1.21.

### v1.1.1
Fixes:
- Elasticsearch and Keycloak images were updated to address CVEs.
- Updated WebLogic Kubernetes Operator version to 3.3.7.
- Minor bug fixes including updating Elasticsearch logging to avoid type collisions.
- Improved cluster-dump behavior when capturing logs.
- Rancher namespace is now created by default.

Known Issues:
- Importing a Kubernetes v1.21 cluster into Rancher might not work properly. Rancher does not currently support Kubernetes v1.21.

### v1.1.0
Fixes:
- Added support for Kiali.
- Simplified the placement of multicluster resources.
- Improved the performance of installing Verrazzano.
- Added support for external Elasticsearch.
- Improvements to system functions, including the authenticating proxy.
- Added support in the LoggingTrait to customize application logging.
- Fixed ability to register a managed cluster with Rancher when configured to use LetsEncrypt staging certificates.
- Fixed Elasticsearch status yellow due to unassigned shards.
- Added support for Kubernetes 1.21, dropped support of Kubernetes 1.18.
- Updated several installed and supported [Software Versions]({{< relref "/docs/setup/prereqs.md" >}}).

Known Issues:
- Importing a Kubernetes v1.21 cluster into Rancher might not work properly. Rancher does not currently support Kubernetes v1.21.

### v1.0.4
Fixes:
- Elasticsearch and Spring Boot images were updated to consume log4j 2.16, to address CVE-2021-44228/CVE-2021-45046.
- Keycloak image was updated to address vulnerabilities.
- Minor bug fixes including fixes for capitalization in user-visible messages.

### v1.0.3
Fixes:
- Fix to use load balancer service external IP address for application ingress when using an external load balancer and wildcard DNS.
- Fixed scraping of Prometheus metrics for WebLogic workloads on managed clusters.
- Rebuilt several component images to address known issues.
- Updated to the following versions:
    - Grafana 6.7.4.
    - WebLogic Kubernetes Operator 3.3.3.

### v1.0.2
Fixes:
- Updated CoreDNS to version 1.6.2-1.
- Updated Keycloak to version 10.0.2.
- Updated WebLogic Kubernetes Operator to version 3.3.2.
- Updated Oracle Linux image to version 7.9.
- Rebuilt several component images to address known issues.
- Fixes/improvements for the analysis tool, including support for diagnosing load balancer limit reached issues.
- Fixes/improvements for the install/upgrade process, including:
    - Install/upgrade jobs now run in the ``verrazzano-install`` namespace.
    - Added Rancher registration status to the VerrazzanoManagedCluster status.
    - Updated OKE troubleshooting URL in installation log.
    - Fixed ExternalIP handling during Istio install.
- Fixed Elasticsearch status yellow due to unassigned_shards.
- Webhook now disallows multicluster resources that are not in a VerrazzanoProject namespace.

### v1.0.1
Fixes:
- Updated to the following versions:
    - WebLogic Kubernetes Operator v3.3.0.
    - Coherence Operator v3.2.1.
    - In the Analysis Tool, `kubectl` v1.20.6-2.
- Ensured ConfigMaps are deleted during uninstall.
- Fixed logging pattern match issue for OKE Kubernetes v1.20.8 clusters.
- Fixed multicluster log collection for Verrazzano installations using LetsEncrypt certificates.
- Fixed console UI display bugs for multicluster applications.
- Fixed a bug where API keys generated by the Oracle Cloud Infrastructure Console were not working correctly.

### v1.0.0
Features: Updated to Rancher v2.5.9.

### v0.17.0
Features:
- Allow Verrazzano Monitoring Instance (VMI) replicas and memory sizes to be changed during installation for both `dev` and `prod` profiles.
- When installing Verrazzano on OKE, the OKE-specific Fluentd `extraVolumeMounts` configuration is no longer required.
- Updated to WebLogic Kubernetes Operator v3.2.5.

Fixes:
- During uninstall, delete application resources only from namespaces which are managed by Verrazzano.
- During upgrade, honor the APP_OPERATOR_IMAGE override.
- Fixed Keycloak installation failure when Prometheus is disabled.
- Allow empty values for Helm overrides in `config.json`.

### v0.16.0
Features:
- Provided options to configure log volume/mount of the log collector, Fluentd, and pre-configured profiles.
- Automatically enabled metrics and log capture for WebLogic domains deployed in Verrazzano.
- Added security-related data/project YAML files to the Verrazzano Console, under project details.
- Updated to WebLogic Kubernetes Operator v3.2.4.

Fixes:
- Added a fix for default metrics traits not always being injected into the `appconfig`.
- Updated the timestamp in WebLogic application logs so that the time filter can be used in Kibana.
- Corrected the incorrect `podSelector` in the node exporter network policy.
- Fixed the DNS resolution issue due to the missing cluster section of the `coredns configmap`.
- Stability improvements for the platform, tests, and examples.
- Renamed the Elasticsearch fields in a multicluster registration secret to be consistent.

### v0.15.1
Features:
- Allow customization of Elasticsearch node sizes and topology during installation.
- If `runtimeEncryptionSecret`, specified in the WebLogic domain spec, does not already exist, then create it.
- Support overrides of persistent storage configuration for Elasticsearch, Kibana, Prometheus, Grafana, and Keycloak.

Known Issues:
- After upgrade to 0.15.1, for Verrazzano Custom Resource installed on Oracle Cloud Infrastructure Container Engine for Kubernetes (OKE), the Fluentd DaemonSet in the `verrazzano-system` namespace cannot access logs.
  Run following command to patch the Fluentd DaemonSet and correct the issue:
  ```
  kubectl patch -n verrazzano-system ds fluentd --patch '{"spec":{"template":{"spec":{"containers":[{"name": "fluentd","volumeMounts":[{"mountPath":"/u01/data/","name":"extravol0","readOnly":true}]}],"volumes":[{"hostPath":{"path":"/u01/data/","type":""},"name":"extravol0"}]}}}}'
  ```

### v0.15.0
Features:
- Support for private container registries.
- Secured communication between Verrazzano resources using Istio.
- Updated to the following versions:
    - cert-manager v1.2.0.
    - Coherence Operator v3.1.5.
    - WebLogic Kubernetes Operator v3.2.3.
    - Node Exporter v1.0.0.
    - NGINX Ingress Controller v0.46.
    - Fluentd v1.12.3.
- Added network policies for Istio.

Fixes:
- Stability improvements for the platform, tests, and examples.
- Several fixes for scraping Prometheus metrics.
- Several fixes for logging and Elasticsearch.
- Replaced `keycloak.json` with dynamic realm creation.
- Removed the LoggingScope CRD from the Verrazzano API.
- Fixed issues related to multicluster resources being orphaned.

### v0.14.0
Features:
- Multicluster support for Verrazzano. Now you can:
    - Register participating clusters as VerrazzanoManagedClusters.
    - Deploy MutiClusterComponents and MultiClusterApplicationConfigurations.
    - Organize multicluster namespaces as VerrazzanoProjects.
    - Access MultiCluster Components and ApplicationConfigurations in the Verrazzano Console UI.
- Changed default wildcard DNS from xip.io to nip.io.
- Support for OKE clusters with private endpoints.
- Support for network policies. Now you can:
    - Add ingress-NGINX network policies.
    - Add Rancher network policies.
    - Add NetworkPolicy support to Verrazzano projects.
    - Add network policies for Keycloak.
    - Add platform operator network policies.
    - Add network policies for Elasticsearch and Kibana.
    - Set network policies for Verrazzano operators, Console, and API proxy.
    - Add network policies for WebLogic Kubernetes Operator.
- Changes to allow magic DNS provider to be specified (xip.io, nip.io, sslip.io).
- Support service setup for multiple containers.
- Enabled use of self-signed certs with Oracle Cloud Infrastructure DNS.
- Support for setting DeploymentStrategy for VerrazzanoHelidonWorkload.

Fixes:

- Several stability improvements for the platform, tests, and examples.
- Added retries around lookup of Rancher admin user.
- Granted specific privileges instead of `ALL` for Keycloak user in MySQL.
- Disabled the installation of the Verrazzano Console UI on managed clusters.

### v0.13.0
Features:
- `IngressTrait` support for explicit destination host and port.
- Experimental cluster diagnostic tooling.
- Grafana dashboards for `VerrazzanoHelidonWorkload`.
- Now you can update application Fluentd sidecar images following a Verrazzano update.
- Documented Verrazzano specific OAM workload resources.
- Documented Verrazzano hardware requirements and installed software versions.

Fixes:
- `VerrazzanoWebLogicWorkload` and `VerrazzanoCoherenceWorkload` resources now handle updates.
- Now `VerrazzanoHelidonWorkload` supports the use of the `ManualScalarTrait`.
- Now you can delete a `Namespace` containing an `ApplicationConfiguration` resource.
- Fixed frequent restarts of Prometheus during application deployment.
- Made `verrazzano-application-operator` logging more useful and use structured logging.
- Fixed Verrazzano uninstall issues.

### v0.12.0
Features:
- Observability stack now uses Keycloak SSO for authentication.
- Istio sidecars now automatically injected when namespaces labeled `istio-injection=enabled`.
- Support for Helidon applications now defined using `VerrazzanoHelidonWorkload` type.

Fixes:
- Fixed issues where logs were not captured from all containers in workloads with multiple containers.
- Fixed issue where some resources were not cleaned up during uninstall.

### v0.11.0

Features:
- OAM applications are optionally deployed into an Istio service mesh.
- Incremental improvements to user-facing roles.

Fixes:
- Fixed issue with logging when an application has multiple workload types.
- Fixed metrics configuration in Spring Boot example application.

### v0.10.0

**Breaking Changes**:
- Model/binding files removed; now application deployment done exclusively by using Open Application Model (OAM).
- Syntax changes for WebLogic and Coherence OAM workloads, now defined using `VerrazzanoCoherenceWorkload`
  and `VerrazzanoWebLogicWorkload` types.

Features:
- By default, application endpoints now use HTTPs - when using magic DNS, certificates are issued by cluster issuer, when using
  Oracle Cloud Infrastructure DNS certificates are issued using Let's Encrypt, or the end user can provide certificates.
- Updated to Coherence Operator v3.1.3.
- Updates for running Verrazzano on Kubernetes 1.19 and 1.20.
- RBAC roles and role bindings created at installation.
- Added instance information to status of Verrazzano custom resource; can be used to obtain instance URLs.
- Updated to Istio v1.7.3.

Fixes:
- Reduced log level of Elasticsearch; excessive logging could have resulted in filling up disks.

### v0.9.0
- Features:
    - Added platform support for installing Verrazzano on Kind clusters.
    - Log records are indexed from the OAM `appconfig` and `component` definitions using the following pattern: `namespace-appconfig-component`.
    - All system and curated components are now patchable.
    - More updates to Open Application Model (OAM) support.

To enable OAM, when you install Verrazzano, specify the following in the Kubernetes manifest file for the Verrazzano custom resource:

```
spec:
  oam:
    enabled: true
```


### v0.8.0
- Features:
    - Support for two installation profiles, development (`dev`) and production (`prod`).  The production profile, which is the default, provides a 3-node Elasticsearch and persistent storage for the Verrazzano Monitoring Instance (VMI). The development profile provides a single node Elasticsearch and no persistent storage for the VMI.
    - The default behavior has been changed to use the system VMI for all monitoring (applications and Verrazzano components).  It is still possible to customize one of the profiles to enable the original, non-shared VMI mode.
    - Initial support for the Open Application Model (OAM).
- Fixes:
    - Updated to Axios NPM package v0.21.1 to resolve a security vulnerability in the examples code.

### v.0.7.0
- Features:
    - Ability to upgrade an existing Verrazzano installation.
    - Added the Verrazzano Console.
    - Enhanced the structure of the Verrazzano custom resource to allow more configurability.
    - Streamlined the secret usage for Oracle Cloud Infrastructure DNS installations.

- Fixes:
    - Fixed bug where the Verrazzano CR `Certificate.CA` fields were being ignored.
    - Removed secret used for `hello-world`; `hello-world-application` image is now public in ghcr so `ImagePullSecrets` is no longer needed.
    - Fixed [issue #339](https://github.com/verrazzano/verrazzano/issues/339) (PRs [#208](https://github.com/verrazzano/verrazzano-operator/pull/208) & [#210](https://github.com/verrazzano/verrazzano-operator/pull/210).)

### v0.6.0
- Features:
    - In-cluster installer which replaces client-side installation scripts.
    - Added installation profiles; in this release, there are two: production and development.
    - Verrazzano system components now emit JSON structured logs.
- Fixes:
    - Updated Elasticsearch and Kibana versions (elasticsearch:7.6.1-20201130145440-5c76ab1) and (kibana:7.6.1-20201130145840-7717e73).
