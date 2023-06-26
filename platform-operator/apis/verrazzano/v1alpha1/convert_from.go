// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertFrom converts from v1beta1.Verrazzano to v1alpha1.Verrazzano
func (in *Verrazzano) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.Verrazzano)
	if src == nil {
		return nil
	}
	in.ObjectMeta = src.ObjectMeta

	// Convert Spec
	in.Spec.Components = convertComponentsFromV1Beta1(src.Spec.Components)
	in.Spec.Profile = ProfileType(src.Spec.Profile)
	in.Spec.EnvironmentName = src.Spec.EnvironmentName
	in.Spec.Version = src.Spec.Version
	in.Spec.DefaultVolumeSource = src.Spec.DefaultVolumeSource
	in.Spec.VolumeClaimSpecTemplates = convertVoumeClaimTemplatesFromV1Beta1(src.Spec.VolumeClaimSpecTemplates)
	in.Spec.Security = convertSecuritySpecFromV1Beta1(src.Spec.Security)

	// Convert status
	in.Status.State = VzStateType(src.Status.State)
	in.Status.Version = src.Status.Version
	in.Status.Conditions = convertConditionsFromV1Beta1(src.Status.Conditions)
	in.Status.Components = convertComponentStatusMapFromV1Beta1(src.Status.Components)
	in.Status.VerrazzanoInstance = convertVerrazzanoInstanceFromV1Beta1(src.Status.VerrazzanoInstance)
	in.Status.Available = src.Status.Available
	return nil
}

func convertVoumeClaimTemplatesFromV1Beta1(in []v1beta1.VolumeClaimSpecTemplate) []VolumeClaimSpecTemplate {
	var templates []VolumeClaimSpecTemplate
	for _, template := range in {
		templates = append(templates, VolumeClaimSpecTemplate{
			ObjectMeta: template.ObjectMeta,
			Spec:       template.Spec,
		})
	}
	return templates
}

func convertConditionsFromV1Beta1(conditions []v1beta1.Condition) []Condition {
	var out []Condition
	for _, condition := range conditions {
		out = append(out, Condition{
			Type:               ConditionType(condition.Type),
			Status:             condition.Status,
			LastTransitionTime: condition.LastTransitionTime,
			Message:            condition.Message,
		})
	}
	return out
}

func convertComponentStatusMapFromV1Beta1(components v1beta1.ComponentStatusMap) ComponentStatusMap {
	if components == nil {
		return nil
	}
	componentStatusMap := ComponentStatusMap{}
	for component, detail := range components {
		if detail != nil {
			componentStatusMap[component] = &ComponentStatusDetails{
				Name:                     detail.Name,
				Conditions:               convertConditionsFromV1Beta1(detail.Conditions),
				State:                    CompStateType(detail.State),
				Available:                convertAvailabilityFrom(detail.Available),
				Version:                  detail.Version,
				LastReconciledGeneration: detail.LastReconciledGeneration,
				ReconcilingGeneration:    detail.ReconcilingGeneration,
			}
		}
	}
	return componentStatusMap
}

func convertAvailabilityFrom(availability *v1beta1.ComponentAvailability) *ComponentAvailability {
	if availability == nil {
		return nil
	}
	newAvailability := ComponentAvailability(*availability)
	return &newAvailability
}

func convertVerrazzanoInstanceFromV1Beta1(instance *v1beta1.InstanceInfo) *InstanceInfo {
	if instance == nil {
		return nil
	}
	return &InstanceInfo{
		ArgoCDURL:      instance.ArgoCDURL,
		ConsoleURL:     instance.ConsoleURL,
		KeyCloakURL:    instance.KeyCloakURL,
		RancherURL:     instance.RancherURL,
		ElasticURL:     instance.OpenSearchURL,
		KibanaURL:      instance.OpenSearchDashboardsURL,
		GrafanaURL:     instance.GrafanaURL,
		PrometheusURL:  instance.PrometheusURL,
		KialiURL:       instance.KialiURL,
		JaegerURL:      instance.JaegerURL,
		ThanosQueryURL: instance.ThanosQueryURL,
	}
}

func convertSecuritySpecFromV1Beta1(security v1beta1.SecuritySpec) SecuritySpec {
	return SecuritySpec{
		AdminSubjects:   security.AdminSubjects,
		MonitorSubjects: security.MonitorSubjects,
	}
}

// convertFluentbitOpensearchOutputFromV1Beta1 converts the v1beta1 FluentbitOpensearchOutputComponent to v1alpha1 FluentbitOpensearchOutputComponent
func convertFluentbitOpensearchOutputFromV1Beta1(in *v1beta1.FluentbitOpensearchOutputComponent) *FluentbitOpensearchOutputComponent {
	if in == nil {
		return nil
	}
	return &FluentbitOpensearchOutputComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertComponentsFromV1Beta1(in v1beta1.ComponentSpec) ComponentSpec {
	return ComponentSpec{
		CertManager:               convertCertManagerFromV1Beta1(in.CertManager),
		ClusterIssuer:             convertClusterIssuerFromV1Beta1(in.ClusterIssuer),
		CertManagerWebhookOCI:     convertCertManagerWebhookOCIFromV1Beta1(in.CertManagerWebhookOCI),
		CoherenceOperator:         convertCoherenceOperatorFromV1Beta1(in.CoherenceOperator),
		ApplicationOperator:       convertApplicationOperatorFromV1Beta1(in.ApplicationOperator),
		AuthProxy:                 convertAuthProxyFromV1Beta1(in.AuthProxy),
		OAM:                       convertOAMFromV1Beta1(in.OAM),
		Console:                   convertConsoleFromV1Beta1(in.Console),
		ClusterOperator:           convertClusterOperatorFromV1Beta1(in.ClusterOperator),
		DNS:                       convertDNSFromV1Beta1(in.DNS),
		Elasticsearch:             convertOpenSearchFromV1Beta1(in.OpenSearch),
		Fluentd:                   convertFluentdFromV1Beta1(in.Fluentd),
		FluentOperator:            convertFluentOperatorFromV1Beta1(in.FluentOperator),
		FluentbitOpensearchOutput: convertFluentbitOpensearchOutputFromV1Beta1(in.FluentbitOpensearchOutput),
		Grafana:                   convertGrafanaFromV1Beta1(in.Grafana),
		Ingress:                   convertIngressNGINXFromV1Beta1(in.IngressNGINX),
		Istio:                     convertIstioFromV1Beta1(in.Istio),
		JaegerOperator:            convertJaegerOperatorFromV1Beta1(in.JaegerOperator),
		Kiali:                     convertKialiFromV1Beta1(in.Kiali),
		Keycloak:                  convertKeycloakFromV1Beta1(in.Keycloak),
		Kibana:                    convertOSDFromV1Beta1(in.OpenSearchDashboards),
		KubeStateMetrics:          convertKubeStateMetricsFromV1Beta1(in.KubeStateMetrics),
		MySQLOperator:             convertMySQLOperatorFromV1Beta1(in.MySQLOperator),
		Prometheus:                convertPrometheusFromV1Beta1(in.Prometheus),
		PrometheusAdapter:         convertPrometheusAdapterFromV1Beta1(in.PrometheusAdapter),
		PrometheusNodeExporter:    convertPrometheusNodeExporterFromV1Beta1(in.PrometheusNodeExporter),
		PrometheusOperator:        convertPrometheusOperatorFromV1Beta1(in.PrometheusOperator),
		PrometheusPushgateway:     convertPrometheusPushGatewayFromV1Beta1(in.PrometheusPushgateway),
		Rancher:                   convertRancherFromV1Beta1(in.Rancher),
		RancherBackup:             convertRancherBackupFromV1Beta1(in.RancherBackup),
		Thanos:                    convertThanosFromV1Beta1(in.Thanos),
		WebLogicOperator:          convertWeblogicOperatorFromV1Beta1(in.WebLogicOperator),
		Velero:                    convertVeleroFromV1Beta1(in.Velero),
		Verrazzano:                convertVerrazzanoFromV1Beta1(in.Verrazzano),
		ArgoCD:                    convertArgoCDFromV1Beta1(in.ArgoCD),
		ClusterAPI:                convertClusterAPIFromV1Beta1(in.ClusterAPI),
		ClusterAgent:              convertClusterAgentFromV1Beta1(in.ClusterAgent),
	}
}

func convertClusterIssuerFromV1Beta1(in *v1beta1.ClusterIssuerComponent) *ClusterIssuerComponent {
	if in == nil {
		return nil
	}
	return &ClusterIssuerComponent{
		Enabled:                  in.Enabled,
		ClusterResourceNamespace: in.ClusterResourceNamespace,
		IssuerConfig:             convertIssuerConfigFromV1Beta1(in.IssuerConfig),
	}
}

func convertIssuerConfigFromV1Beta1(in v1beta1.IssuerConfig) IssuerConfig {
	var leIssuer *LetsEncryptACMEIssuer
	if in.LetsEncrypt != nil {
		leIssuer = &LetsEncryptACMEIssuer{
			EmailAddress: in.LetsEncrypt.EmailAddress,
			Environment:  in.LetsEncrypt.Environment,
		}
	}
	var caIssuer *CAIssuer
	if in.CA != nil {
		caIssuer = &CAIssuer{SecretName: in.CA.SecretName}
	}
	return IssuerConfig{
		LetsEncrypt: leIssuer,
		CA:          caIssuer,
	}
}

func convertCertManagerWebhookOCIFromV1Beta1(in *v1beta1.CertManagerWebhookOCIComponent) *CertManagerWebhookOCIComponent {
	if in == nil {
		return nil
	}
	return &CertManagerWebhookOCIComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertApplicationOperatorFromV1Beta1(in *v1beta1.ApplicationOperatorComponent) *ApplicationOperatorComponent {
	if in == nil {
		return nil
	}
	return &ApplicationOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertAuthProxyFromV1Beta1(in *v1beta1.AuthProxyComponent) *AuthProxyComponent {
	if in == nil {
		return nil
	}
	return &AuthProxyComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertClusterAPIFromV1Beta1(in *v1beta1.ClusterAPIComponent) *ClusterAPIComponent {
	if in == nil {
		return nil
	}
	return &ClusterAPIComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertClusterAgentFromV1Beta1(in *v1beta1.ClusterAgentComponent) *ClusterAgentComponent {
	if in == nil {
		return nil
	}
	return &ClusterAgentComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertCertManagerFromV1Beta1(in *v1beta1.CertManagerComponent) *CertManagerComponent {
	if in == nil {
		return nil
	}
	return &CertManagerComponent{
		Certificate:      convertCertificateFromV1Beta1(in.Certificate),
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertCertificateFromV1Beta1(in v1beta1.Certificate) Certificate {
	return Certificate{
		Acme: Acme{
			EmailAddress: in.Acme.EmailAddress,
			Environment:  in.Acme.Environment,
			Provider:     ProviderType(in.Acme.Provider),
		},
		CA: CA{
			ClusterResourceNamespace: in.CA.ClusterResourceNamespace,
			SecretName:               in.CA.SecretName,
		},
	}
}

func convertCoherenceOperatorFromV1Beta1(in *v1beta1.CoherenceOperatorComponent) *CoherenceOperatorComponent {
	if in == nil {
		return nil
	}
	return &CoherenceOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertConsoleFromV1Beta1(in *v1beta1.ConsoleComponent) *ConsoleComponent {
	if in == nil {
		return nil
	}
	return &ConsoleComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertOpenSearchFromV1Beta1(in *v1beta1.OpenSearchComponent) *ElasticsearchComponent {
	if in == nil {
		return nil
	}
	return &ElasticsearchComponent{
		Enabled:              in.Enabled,
		Policies:             in.Policies,
		Nodes:                convertOSNodesFromV1Beta1(in.Nodes),
		Plugins:              in.Plugins,
		DisableDefaultPolicy: in.DisableDefaultPolicy,
	}
}

func convertOSNodesFromV1Beta1(in []v1beta1.OpenSearchNode) []OpenSearchNode {
	var out []OpenSearchNode
	for _, inNode := range in {
		var storage *OpenSearchNodeStorage
		if inNode.Storage != nil {
			storage = &OpenSearchNodeStorage{
				Size: inNode.Storage.Size,
			}
		}
		out = append(out, OpenSearchNode{
			Name:      inNode.Name,
			Replicas:  inNode.Replicas,
			Roles:     inNode.Roles,
			Storage:   storage,
			Resources: inNode.Resources,
			JavaOpts:  inNode.JavaOpts,
		})
	}
	return out
}

func convertDNSFromV1Beta1(in *v1beta1.DNSComponent) *DNSComponent {
	if in == nil {
		return nil
	}
	return &DNSComponent{
		Wildcard:         convertWildcardDNSFromV1Beta1(in.Wildcard),
		OCI:              convertOCIDNSFromV1Beta1(in.OCI),
		External:         convertExternalDNSFromV1Beta1(in.External),
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertWildcardDNSFromV1Beta1(wildcard *v1beta1.Wildcard) *Wildcard {
	if wildcard == nil {
		return nil
	}
	return &Wildcard{
		Domain: wildcard.Domain,
	}
}

func convertOCIDNSFromV1Beta1(oci *v1beta1.OCI) *OCI {
	if oci == nil {
		return nil
	}
	return &OCI{
		OCIConfigSecret:        oci.OCIConfigSecret,
		DNSZoneCompartmentOCID: oci.DNSZoneCompartmentOCID,
		DNSZoneOCID:            oci.DNSZoneOCID,
		DNSZoneName:            oci.DNSZoneName,
		DNSScope:               oci.DNSScope,
	}
}

func convertExternalDNSFromV1Beta1(external *v1beta1.External) *External {
	if external == nil {
		return nil
	}
	return &External{Suffix: external.Suffix}
}

func convertFluentdFromV1Beta1(in *v1beta1.FluentdComponent) *FluentdComponent {
	if in == nil {
		return nil
	}
	return &FluentdComponent{
		Enabled:             in.Enabled,
		ExtraVolumeMounts:   convertVolumeMountsFromV1Beta1(in.ExtraVolumeMounts),
		ElasticsearchURL:    in.OpenSearchURL,
		ElasticsearchSecret: in.OpenSearchSecret,
		OCI:                 convertOCILoggingConfigurationFromV1Beta1(in.OCI),
		InstallOverrides:    convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

// // convertFluentOperatorFromV1Beta1 converts the v1beta1 FluentOperatorComponent to v1alpha1 FluentOperatorComponent
func convertFluentOperatorFromV1Beta1(in *v1beta1.FluentOperatorComponent) *FluentOperatorComponent {
	if in == nil {
		return nil
	}
	return &FluentOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertVolumeMountsFromV1Beta1(mounts []v1beta1.VolumeMount) []VolumeMount {
	var out []VolumeMount
	for _, mount := range mounts {
		out = append(out, VolumeMount{
			Source:      mount.Source,
			Destination: mount.Destination,
			ReadOnly:    mount.ReadOnly,
		})
	}
	return out
}

func convertOCILoggingConfigurationFromV1Beta1(oci *v1beta1.OciLoggingConfiguration) *OciLoggingConfiguration {
	if oci == nil {
		return nil
	}
	return &OciLoggingConfiguration{
		DefaultAppLogID: oci.DefaultAppLogID,
		SystemLogID:     oci.SystemLogID,
		APISecret:       oci.APISecret,
	}
}

func convertGrafanaFromV1Beta1(in *v1beta1.GrafanaComponent) *GrafanaComponent {
	if in == nil {
		return nil
	}
	var info *DatabaseInfo
	if in.Database != nil {
		info = &DatabaseInfo{
			Host: in.Database.Host,
			Name: in.Database.Name,
		}
	}
	return &GrafanaComponent{
		Enabled:  in.Enabled,
		Replicas: in.Replicas,
		Database: info,
		SMTP:     in.SMTP,
	}
}

func convertIngressNGINXFromV1Beta1(in *v1beta1.IngressNginxComponent) *IngressNginxComponent {
	if in == nil {
		return nil
	}
	return &IngressNginxComponent{
		IngressClassName: in.IngressClassName,
		Type:             IngressType(in.Type),
		Ports:            in.Ports,
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertIstioFromV1Beta1(in *v1beta1.IstioComponent) *IstioComponent {
	if in == nil {
		return nil
	}
	return &IstioComponent{
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
		Enabled:          in.Enabled,
		InjectionEnabled: in.InjectionEnabled,
	}
}

func convertJaegerOperatorFromV1Beta1(in *v1beta1.JaegerOperatorComponent) *JaegerOperatorComponent {
	if in == nil {
		return nil
	}
	return &JaegerOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertKialiFromV1Beta1(in *v1beta1.KialiComponent) *KialiComponent {
	if in == nil {
		return nil
	}
	return &KialiComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertKeycloakFromV1Beta1(in *v1beta1.KeycloakComponent) *KeycloakComponent {
	if in == nil {
		return nil
	}
	return &KeycloakComponent{
		MySQL: MySQLComponent{
			VolumeSource:     in.MySQL.VolumeSource,
			InstallOverrides: convertInstallOverridesFromV1Beta1(in.MySQL.InstallOverrides),
		},
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertOAMFromV1Beta1(in *v1beta1.OAMComponent) *OAMComponent {
	if in == nil {
		return nil
	}
	return &OAMComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertOSDFromV1Beta1(in *v1beta1.OpenSearchDashboardsComponent) *KibanaComponent {
	if in == nil {
		return nil
	}
	return &KibanaComponent{
		Enabled:  in.Enabled,
		Replicas: in.Replicas,
		Plugins:  in.Plugins,
	}
}

func convertKubeStateMetricsFromV1Beta1(in *v1beta1.KubeStateMetricsComponent) *KubeStateMetricsComponent {
	if in == nil {
		return nil
	}
	return &KubeStateMetricsComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertMySQLOperatorFromV1Beta1(in *v1beta1.MySQLOperatorComponent) *MySQLOperatorComponent {
	if in == nil {
		return nil
	}
	return &MySQLOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertPrometheusFromV1Beta1(in *v1beta1.PrometheusComponent) *PrometheusComponent {
	if in == nil {
		return nil
	}
	return &PrometheusComponent{
		Enabled: in.Enabled,
	}
}

func convertPrometheusAdapterFromV1Beta1(in *v1beta1.PrometheusAdapterComponent) *PrometheusAdapterComponent {
	if in == nil {
		return nil
	}
	return &PrometheusAdapterComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertPrometheusNodeExporterFromV1Beta1(in *v1beta1.PrometheusNodeExporterComponent) *PrometheusNodeExporterComponent {
	if in == nil {
		return nil
	}
	return &PrometheusNodeExporterComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertPrometheusOperatorFromV1Beta1(in *v1beta1.PrometheusOperatorComponent) *PrometheusOperatorComponent {
	if in == nil {
		return nil
	}
	return &PrometheusOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertPrometheusPushGatewayFromV1Beta1(in *v1beta1.PrometheusPushgatewayComponent) *PrometheusPushgatewayComponent {
	if in == nil {
		return nil
	}
	return &PrometheusPushgatewayComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertRancherFromV1Beta1(in *v1beta1.RancherComponent) *RancherComponent {
	if in == nil {
		return nil
	}
	return &RancherComponent{
		Enabled:             in.Enabled,
		InstallOverrides:    convertInstallOverridesFromV1Beta1(in.InstallOverrides),
		KeycloakAuthEnabled: in.KeycloakAuthEnabled,
	}
}

func convertRancherBackupFromV1Beta1(in *v1beta1.RancherBackupComponent) *RancherBackupComponent {
	if in == nil {
		return nil
	}
	return &RancherBackupComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertWeblogicOperatorFromV1Beta1(in *v1beta1.WebLogicOperatorComponent) *WebLogicOperatorComponent {
	if in == nil {
		return nil
	}
	return &WebLogicOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertVeleroFromV1Beta1(in *v1beta1.VeleroComponent) *VeleroComponent {
	if in == nil {
		return nil
	}
	return &VeleroComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertVerrazzanoFromV1Beta1(in *v1beta1.VerrazzanoComponent) *VerrazzanoComponent {
	if in == nil {
		return nil
	}
	return &VerrazzanoComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertArgoCDFromV1Beta1(in *v1beta1.ArgoCDComponent) *ArgoCDComponent {
	if in == nil {
		return nil
	}
	return &ArgoCDComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertClusterOperatorFromV1Beta1(in *v1beta1.ClusterOperatorComponent) *ClusterOperatorComponent {
	if in == nil {
		return nil
	}
	return &ClusterOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(in.InstallOverrides),
	}
}

func convertThanosFromV1Beta1(src *v1beta1.ThanosComponent) *ThanosComponent {
	if src == nil {
		return nil
	}
	return &ThanosComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFromV1Beta1(src.InstallOverrides),
	}
}

func convertInstallOverridesFromV1Beta1(in v1beta1.InstallOverrides) InstallOverrides {
	return InstallOverrides{
		MonitorChanges: in.MonitorChanges,
		ValueOverrides: convertValueOverridesFromV1Beta1(in.ValueOverrides),
	}
}

func convertValueOverridesFromV1Beta1(in []v1beta1.Overrides) []Overrides {
	var out []Overrides
	for _, oIn := range in {
		out = append(out, Overrides{
			ConfigMapRef: oIn.ConfigMapRef,
			SecretRef:    oIn.SecretRef,
			Values:       oIn.Values,
		})
	}
	return out
}
