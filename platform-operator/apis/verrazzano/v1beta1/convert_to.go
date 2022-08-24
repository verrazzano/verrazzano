// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (in *Verrazzano) ConvertTo(dstRaw conversion.Hub) error {
	out := dstRaw.(*v1alpha1.Verrazzano)
	out.ObjectMeta = in.ObjectMeta

	// Convert Spec
	out.Spec.Profile = v1alpha1.ProfileType(in.Spec.Profile)
	out.Spec.EnvironmentName = in.Spec.EnvironmentName
	out.Spec.Version = in.Spec.Version
	out.Spec.DefaultVolumeSource = in.Spec.DefaultVolumeSource
	out.Spec.Security = convertSecuritySpecTo(in.Spec.Security)
	out.Spec.Components = convertComponentsTo(in.Spec.Components)

	// Convert Status
	out.Status.State = v1alpha1.VzStateType(in.Status.State)
	out.Status.Version = in.Status.Version
	out.Status.Conditions = convertConditionsTo(in.Status.Conditions)
	out.Status.Components = convertComponentStatusMapTo(in.Status.Components)
	out.Status.VerrazzanoInstance = convertVerrazzanoInstanceTo(in.Status.VerrazzanoInstance)
	return nil
}

func convertConditionsTo(conditions []Condition) []v1alpha1.Condition {
	var out []v1alpha1.Condition
	for _, condition := range conditions {
		out = append(out, v1alpha1.Condition{
			Type:               v1alpha1.ConditionType(condition.Type),
			Status:             condition.Status,
			LastTransitionTime: condition.LastTransitionTime,
			Message:            condition.Message,
		})
	}
	return out
}

func convertComponentStatusMapTo(components ComponentStatusMap) v1alpha1.ComponentStatusMap {
	if components == nil {
		return nil
	}
	componentStatusMap := v1alpha1.ComponentStatusMap{}
	for component, detail := range components {
		if detail != nil {
			componentStatusMap[component] = &v1alpha1.ComponentStatusDetails{
				Name:                     detail.Name,
				Conditions:               convertConditionsTo(detail.Conditions),
				State:                    v1alpha1.CompStateType(detail.State),
				Version:                  detail.Version,
				LastReconciledGeneration: detail.LastReconciledGeneration,
				ReconcilingGeneration:    detail.ReconcilingGeneration,
			}
		}
	}
	return componentStatusMap
}

func convertVerrazzanoInstanceTo(instance *InstanceInfo) *v1alpha1.InstanceInfo {
	if instance == nil {
		return nil
	}
	return &v1alpha1.InstanceInfo{
		ConsoleURL:    instance.ConsoleURL,
		KeyCloakURL:   instance.KeyCloakURL,
		RancherURL:    instance.RancherURL,
		ElasticURL:    instance.OpenSearchURL,
		KibanaURL:     instance.OpenSearchDashboardsURL,
		GrafanaURL:    instance.GrafanaURL,
		PrometheusURL: instance.PrometheusURL,
		KialiURL:      instance.KialiURL,
		JaegerURL:     instance.JaegerURL,
	}
}

func convertSecuritySpecTo(security SecuritySpec) v1alpha1.SecuritySpec {
	return v1alpha1.SecuritySpec{
		AdminSubjects:   security.AdminSubjects,
		MonitorSubjects: security.MonitorSubjects,
	}
}

func convertComponentsTo(in ComponentSpec) v1alpha1.ComponentSpec {
	return v1alpha1.ComponentSpec{
		CertManager:            convertCertManagerTo(in.CertManager),
		CoherenceOperator:      convertCoherenceOperatorTo(in.CoherenceOperator),
		ApplicationOperator:    convertApplicationOperatorTo(in.ApplicationOperator),
		AuthProxy:              convertAuthProxyTo(in.AuthProxy),
		OAM:                    convertOAMTo(in.OAM),
		Console:                convertConsoleTo(in.Console),
		DNS:                    convertDNSTo(in.DNS),
		Elasticsearch:          convertOpenSearchTo(in.OpenSearch),
		Fluentd:                convertFluentdTo(in.Fluentd),
		Grafana:                convertGrafanaTo(in.Grafana),
		Ingress:                convertIngressNGINXTo(in.Ingress),
		Istio:                  convertIstioTo(in.Istio),
		JaegerOperator:         convertJaegerOperatorTo(in.JaegerOperator),
		Kiali:                  convertKialiTo(in.Kiali),
		Keycloak:               convertKeycloakTo(in.Keycloak),
		Kibana:                 convertOSDTo(in.OpenSearchDashboards),
		KubeStateMetrics:       convertKubeStateMetricsTo(in.KubeStateMetrics),
		MySQLOperator:          convertMySQLOperatorTo(in.MySQLOperator),
		Prometheus:             convertPrometheusTo(in.Prometheus),
		PrometheusAdapter:      convertPrometheusAdapterTo(in.PrometheusAdapter),
		PrometheusNodeExporter: convertPrometheusNodeExporterTo(in.PrometheusNodeExporter),
		PrometheusOperator:     convertPrometheusOperatorTo(in.PrometheusOperator),
		PrometheusPushgateway:  convertPrometheusPushGatewayTo(in.PrometheusPushgateway),
		Rancher:                convertRancherTo(in.Rancher),
		RancherBackup:          convertRancherBackupTo(in.RancherBackup),
		WebLogicOperator:       convertWeblogicOperatorTo(in.WebLogicOperator),
		Velero:                 convertVeleroTo(in.Velero),
		Verrazzano:             convertVerrazzanoTo(in.Verrazzano),
	}
}

func convertApplicationOperatorTo(in *ApplicationOperatorComponent) *v1alpha1.ApplicationOperatorComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.ApplicationOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertAuthProxyTo(in *AuthProxyComponent) *v1alpha1.AuthProxyComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.AuthProxyComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertCertManagerTo(in *CertManagerComponent) *v1alpha1.CertManagerComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.CertManagerComponent{
		Certificate:      convertCertificateTo(in.Certificate),
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertCertificateTo(certificate Certificate) v1alpha1.Certificate {
	return v1alpha1.Certificate{
		Acme: v1alpha1.Acme{
			Provider:     v1alpha1.ProviderType(certificate.Acme.Provider),
			EmailAddress: certificate.Acme.EmailAddress,
			Environment:  certificate.Acme.Environment,
		},
		CA: v1alpha1.CA{
			SecretName:               certificate.CA.SecretName,
			ClusterResourceNamespace: certificate.CA.ClusterResourceNamespace,
		},
	}
}

func convertCoherenceOperatorTo(in *CoherenceOperatorComponent) *v1alpha1.CoherenceOperatorComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.CoherenceOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertConsoleTo(in *ConsoleComponent) *v1alpha1.ConsoleComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.ConsoleComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertOpenSearchTo(in *OpenSearchComponent) *v1alpha1.ElasticsearchComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.ElasticsearchComponent{
		Enabled:  in.Enabled,
		Policies: in.Policies,
		Nodes:    convertOSNodesTo(in.Nodes),
	}
}

func convertOSNodesTo(in []OpenSearchNode) []v1alpha1.OpenSearchNode {
	var out []v1alpha1.OpenSearchNode
	for _, inNode := range in {
		var storage *v1alpha1.OpenSearchNodeStorage
		if inNode.Storage != nil {
			storage = &v1alpha1.OpenSearchNodeStorage{
				Size: inNode.Storage.Size,
			}
		}
		out = append(out, v1alpha1.OpenSearchNode{
			Name:      inNode.Name,
			Replicas:  inNode.Replicas,
			Roles:     inNode.Roles,
			Storage:   storage,
			Resources: inNode.Resources,
		})
	}
	return out
}

func convertDNSTo(in *DNSComponent) *v1alpha1.DNSComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.DNSComponent{
		Wildcard:         convertWildcardDNSTo(in.Wildcard),
		OCI:              convertOCIDNSTo(in.OCI),
		External:         convertExternalDNSTo(in.External),
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertWildcardDNSTo(wildcard *Wildcard) *v1alpha1.Wildcard {
	if wildcard == nil {
		return nil
	}
	return &v1alpha1.Wildcard{
		Domain: wildcard.Domain,
	}
}

func convertOCIDNSTo(oci *OCI) *v1alpha1.OCI {
	if oci == nil {
		return nil
	}
	return &v1alpha1.OCI{
		OCIConfigSecret:        oci.OCIConfigSecret,
		DNSZoneCompartmentOCID: oci.DNSZoneCompartmentOCID,
		DNSZoneOCID:            oci.DNSZoneOCID,
		DNSZoneName:            oci.DNSZoneName,
		DNSScope:               oci.DNSScope,
	}
}

func convertExternalDNSTo(external *External) *v1alpha1.External {
	if external == nil {
		return nil
	}
	return &v1alpha1.External{Suffix: external.Suffix}
}

func convertFluentdTo(in *FluentdComponent) *v1alpha1.FluentdComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.FluentdComponent{
		Enabled:             in.Enabled,
		ExtraVolumeMounts:   convertVolumeMountsTo(in.ExtraVolumeMounts),
		ElasticsearchURL:    in.OpenSearchURL,
		ElasticsearchSecret: in.OpenSearchSecret,
		OCI:                 convertOCILoggingConfigurationTo(in.OCI),
		InstallOverrides:    convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertVolumeMountsTo(mounts []VolumeMount) []v1alpha1.VolumeMount {
	var out []v1alpha1.VolumeMount
	for _, mount := range mounts {
		out = append(out, v1alpha1.VolumeMount{
			Source:      mount.Source,
			Destination: mount.Destination,
			ReadOnly:    mount.ReadOnly,
		})
	}
	return out
}

func convertOCILoggingConfigurationTo(oci *OciLoggingConfiguration) *v1alpha1.OciLoggingConfiguration {
	if oci == nil {
		return nil
	}
	return &v1alpha1.OciLoggingConfiguration{
		DefaultAppLogID: oci.DefaultAppLogID,
		SystemLogID:     oci.SystemLogID,
		APISecret:       oci.APISecret,
	}
}

func convertGrafanaTo(in *GrafanaComponent) *v1alpha1.GrafanaComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.GrafanaComponent{
		Enabled:  in.Enabled,
		Replicas: in.Replicas,
		Database: &v1alpha1.DatabaseInfo{
			Host: in.Database.Host,
			Name: in.Database.Name,
		},
	}
}

func convertIngressNGINXTo(in *IngressNginxComponent) *v1alpha1.IngressNginxComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.IngressNginxComponent{
		IngressClassName: in.IngressClassName,
		Type:             v1alpha1.IngressType(in.Type),
		Ports:            in.Ports,
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertIstioTo(in *IstioComponent) *v1alpha1.IstioComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.IstioComponent{
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
		Enabled:          in.Enabled,
		InjectionEnabled: in.InjectionEnabled,
	}
}

func convertJaegerOperatorTo(in *JaegerOperatorComponent) *v1alpha1.JaegerOperatorComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.JaegerOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertKialiTo(in *KialiComponent) *v1alpha1.KialiComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.KialiComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertKeycloakTo(in *KeycloakComponent) *v1alpha1.KeycloakComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.KeycloakComponent{
		MySQL: v1alpha1.MySQLComponent{
			VolumeSource:     in.MySQL.VolumeSource,
			InstallOverrides: convertInstallOverridesTo(in.MySQL.InstallOverrides),
		},
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertMySQLOperatorTo(in *MySQLOperatorComponent) *v1alpha1.MySQLOperatorComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.MySQLOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertOAMTo(in *OAMComponent) *v1alpha1.OAMComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.OAMComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertOSDTo(in *OpenSearchDashboardsComponent) *v1alpha1.KibanaComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.KibanaComponent{
		Enabled:  in.Enabled,
		Replicas: in.Replicas,
	}
}

func convertKubeStateMetricsTo(in *KubeStateMetricsComponent) *v1alpha1.KubeStateMetricsComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.KubeStateMetricsComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertPrometheusTo(in *PrometheusComponent) *v1alpha1.PrometheusComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.PrometheusComponent{
		Enabled: in.Enabled,
	}
}

func convertPrometheusAdapterTo(in *PrometheusAdapterComponent) *v1alpha1.PrometheusAdapterComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.PrometheusAdapterComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertPrometheusNodeExporterTo(in *PrometheusNodeExporterComponent) *v1alpha1.PrometheusNodeExporterComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.PrometheusNodeExporterComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertPrometheusOperatorTo(in *PrometheusOperatorComponent) *v1alpha1.PrometheusOperatorComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.PrometheusOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertPrometheusPushGatewayTo(in *PrometheusPushgatewayComponent) *v1alpha1.PrometheusPushgatewayComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.PrometheusPushgatewayComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertRancherTo(in *RancherComponent) *v1alpha1.RancherComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.RancherComponent{
		Enabled:             in.Enabled,
		InstallOverrides:    convertInstallOverridesTo(in.InstallOverrides),
		KeycloakAuthEnabled: in.KeycloakAuthEnabled,
	}
}

func convertRancherBackupTo(in *RancherBackupComponent) *v1alpha1.RancherBackupComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.RancherBackupComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertWeblogicOperatorTo(in *WebLogicOperatorComponent) *v1alpha1.WebLogicOperatorComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.WebLogicOperatorComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertVeleroTo(in *VeleroComponent) *v1alpha1.VeleroComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.VeleroComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertVerrazzanoTo(in *VerrazzanoComponent) *v1alpha1.VerrazzanoComponent {
	if in == nil {
		return nil
	}
	return &v1alpha1.VerrazzanoComponent{
		Enabled:          in.Enabled,
		InstallOverrides: convertInstallOverridesTo(in.InstallOverrides),
	}
}

func convertInstallOverridesTo(in InstallOverrides) v1alpha1.InstallOverrides {
	return v1alpha1.InstallOverrides{
		MonitorChanges: in.MonitorChanges,
		ValueOverrides: convertValueOverridesTo(in.ValueOverrides),
	}
}

func convertValueOverridesTo(in []Overrides) []v1alpha1.Overrides {
	var out []v1alpha1.Overrides
	for _, oIn := range in {
		out = append(out, v1alpha1.Overrides{
			ConfigMapRef: oIn.ConfigMapRef,
			SecretRef:    oIn.SecretRef,
			Values:       oIn.Values,
		})
	}
	return out
}
