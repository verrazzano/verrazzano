// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	"sigs.k8s.io/yaml"
	"strconv"
)

const (
	masterNodeReplicas = "nodes.master.replicas"
	masterNodeMemory   = "nodes.master.requests.memory"
	masterNodeStorage  = "nodes.master.requests.storage"
	ingestNodeReplicas = "nodes.ingest.replicas"
	ingestNodeMemory   = "nodes.ingest.requests.memory"
	dataNodeReplicas   = "nodes.data.replicas"
	dataNodeMemory     = "nodes.data.requests.memory"
	dataNodeStorage    = "nodes.data.requests.storage"

	masterNodeName = "es-master"
	dataNodeName   = "es-data"
	ingestNodeName = "es-ingest"

	authProxyReplicasKey = "replicas"
	authProxyAffinityKey = "affinity"
)

type expandInfo struct {
	leftMargin int
	key        string
}

//ConvertTo converts a v1alpha1.Verrazzano to a v1beta1.Verrazzano
func (in *Verrazzano) ConvertTo(dstRaw conversion.Hub) error {
	out := dstRaw.(*v1beta1.Verrazzano)
	if out == nil || in == nil {
		return nil
	}
	out.ObjectMeta = in.ObjectMeta

	components, err := convertComponentsTo(in.Spec.Components)
	if err != nil {
		return err
	}

	// Convert Spec
	out.Spec.Profile = v1beta1.ProfileType(in.Spec.Profile)
	out.Spec.EnvironmentName = in.Spec.EnvironmentName
	out.Spec.Version = in.Spec.Version
	out.Spec.DefaultVolumeSource = in.Spec.DefaultVolumeSource
	out.Spec.VolumeClaimSpecTemplates = convertVolumeClaimTemplateTo(in.Spec.VolumeClaimSpecTemplates)
	out.Spec.Components = components
	out.Spec.Security = convertSecuritySpecTo(in.Spec.Security)

	// Convert Status
	out.Status.State = v1beta1.VzStateType(in.Status.State)
	out.Status.Version = in.Status.Version
	out.Status.Conditions = convertConditionsTo(in.Status.Conditions)
	out.Status.Components = convertComponentStatusMapTo(in.Status.Components)
	out.Status.VerrazzanoInstance = convertVerrazzanoInstanceTo(in.Status.VerrazzanoInstance)
	return nil
}

func convertVolumeClaimTemplateTo(src []VolumeClaimSpecTemplate) []v1beta1.VolumeClaimSpecTemplate {
	var templates []v1beta1.VolumeClaimSpecTemplate
	for _, template := range src {
		templates = append(templates, v1beta1.VolumeClaimSpecTemplate{
			ObjectMeta: template.ObjectMeta,
			Spec:       template.Spec,
		})
	}
	return templates
}

func convertComponentsTo(src ComponentSpec) (v1beta1.ComponentSpec, error) {
	authProxyComponent, err := convertAuthProxyToV1Beta1(src.AuthProxy)
	if err != nil {
		return v1beta1.ComponentSpec{}, err
	}
	opensearchComponent, err := convertOpenSearchToV1Beta1(src.Elasticsearch)
	if err != nil {
		return v1beta1.ComponentSpec{}, err
	}
	ingressComponent, err := convertIngressNGINXToV1Beta1(src.Ingress)
	if err != nil {
		return v1beta1.ComponentSpec{}, err
	}
	istioComponent, err := convertIstioToV1Beta1(src.Istio)
	if err != nil {
		return v1beta1.ComponentSpec{}, err
	}
	keycloakComponent, err := convertKeycloakToV1Beta1(src.Keycloak)
	if err != nil {
		return v1beta1.ComponentSpec{}, err
	}
	verrazzanoComponent, err := convertVerrazzanoToV1Beta1(src.Verrazzano)
	if err != nil {
		return v1beta1.ComponentSpec{}, err
	}
	return v1beta1.ComponentSpec{
		CertManager:            convertCertManagerToV1Beta1(src.CertManager),
		CoherenceOperator:      convertCoherenceOperatorToV1Beta1(src.CoherenceOperator),
		ApplicationOperator:    convertApplicationOperatorToV1Beta1(src.ApplicationOperator),
		AuthProxy:              authProxyComponent,
		OAM:                    convertOAMToV1Beta1(src.OAM),
		Console:                convertConsoleToV1Beta1(src.Console),
		DNS:                    convertDNSToV1Beta1(src.DNS),
		OpenSearch:             opensearchComponent,
		Fluentd:                convertFluentdToV1Beta1(src.Fluentd),
		Grafana:                convertGrafanaToV1Beta1(src.Grafana),
		Ingress:                ingressComponent,
		Istio:                  istioComponent,
		JaegerOperator:         convertJaegerOperatorToV1Beta1(src.JaegerOperator),
		Kiali:                  convertKialiToV1Beta1(src.Kiali),
		Keycloak:               keycloakComponent,
		OpenSearchDashboards:   convertOSDToV1Beta1(src.Kibana),
		KubeStateMetrics:       convertKubeStateMetricsToV1Beta1(src.KubeStateMetrics),
		Prometheus:             convertPrometheusToV1Beta1(src.Prometheus),
		PrometheusAdapter:      convertPrometheusAdapterToV1Beta1(src.PrometheusAdapter),
		PrometheusNodeExporter: convertPrometheusNodeExporterToV1Beta1(src.PrometheusNodeExporter),
		PrometheusOperator:     convertPrometheusOperatorToV1Beta1(src.PrometheusOperator),
		PrometheusPushgateway:  convertPrometheusPushGatewayToV1Beta1(src.PrometheusPushgateway),
		Rancher:                convertRancherToV1Beta1(src.Rancher),
		RancherBackup:          convertRancherBackupToV1Beta1(src.RancherBackup),
		WebLogicOperator:       convertWeblogicOperatorToV1Beta1(src.WebLogicOperator),
		Velero:                 convertVeleroToV1Beta1(src.Velero),
		Verrazzano:             verrazzanoComponent,
	}, nil
}

func convertCertManagerToV1Beta1(src *CertManagerComponent) *v1beta1.CertManagerComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.CertManagerComponent{
		Certificate:      convertCertificateToV1Beta1(src.Certificate),
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertCertificateToV1Beta1(certificate Certificate) v1beta1.Certificate {
	return v1beta1.Certificate{
		Acme: v1beta1.Acme{
			Provider:     v1beta1.ProviderType(certificate.Acme.Provider),
			EmailAddress: certificate.Acme.EmailAddress,
			Environment:  certificate.Acme.Environment,
		},
		CA: v1beta1.CA{
			SecretName:               certificate.CA.SecretName,
			ClusterResourceNamespace: certificate.CA.ClusterResourceNamespace,
		},
	}
}

func convertCoherenceOperatorToV1Beta1(src *CoherenceOperatorComponent) *v1beta1.CoherenceOperatorComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.CoherenceOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertApplicationOperatorToV1Beta1(src *ApplicationOperatorComponent) *v1beta1.ApplicationOperatorComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.ApplicationOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertAuthProxyToV1Beta1(src *AuthProxyComponent) (*v1beta1.AuthProxyComponent, error) {
	if src == nil {
		return nil, nil
	}
	overrides := convertInstallOverridesToV1Beta1(src.InstallOverrides)
	if src.Kubernetes != nil {
		replicasInfo := expandInfo{
			key:        authProxyReplicasKey,
			leftMargin: 0,
		}
		affinityInfo := expandInfo{
			key:        authProxyAffinityKey,
			leftMargin: 0,
		}
		k8sSpecYaml, err := convertCommonKubernetesToYaml(src.Kubernetes.CommonKubernetesSpec, replicasInfo, affinityInfo)
		if err != nil {
			return nil, err
		}
		override, err := createValueOverride([]byte(k8sSpecYaml))
		if err != nil {
			return nil, err
		}
		overrides.ValueOverrides = append(overrides.ValueOverrides, override)
	}

	return &v1beta1.AuthProxyComponent{
		Enabled:          src.Enabled,
		InstallOverrides: overrides,
	}, nil
}

func convertOAMToV1Beta1(src *OAMComponent) *v1beta1.OAMComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.OAMComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertConsoleToV1Beta1(src *ConsoleComponent) *v1beta1.ConsoleComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.ConsoleComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertDNSToV1Beta1(src *DNSComponent) *v1beta1.DNSComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.DNSComponent{
		Wildcard:         convertWildcardDNSToV1Beta1(src.Wildcard),
		OCI:              convertOCIDNSToV1Beta1(src.OCI),
		External:         convertExternalDNSToV1Beta1(src.External),
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertWildcardDNSToV1Beta1(wildcard *Wildcard) *v1beta1.Wildcard {
	if wildcard == nil {
		return nil
	}
	return &v1beta1.Wildcard{
		Domain: wildcard.Domain,
	}
}

func convertOCIDNSToV1Beta1(oci *OCI) *v1beta1.OCI {
	if oci == nil {
		return nil
	}
	return &v1beta1.OCI{
		OCIConfigSecret:        oci.OCIConfigSecret,
		DNSZoneCompartmentOCID: oci.DNSZoneCompartmentOCID,
		DNSZoneOCID:            oci.DNSZoneOCID,
		DNSZoneName:            oci.DNSZoneName,
		DNSScope:               oci.DNSScope,
	}
}

func convertExternalDNSToV1Beta1(external *External) *v1beta1.External {
	if external == nil {
		return nil
	}
	return &v1beta1.External{Suffix: external.Suffix}
}

func convertOpenSearchToV1Beta1(src *ElasticsearchComponent) (*v1beta1.OpenSearchComponent, error) {
	if src == nil {
		return nil, nil
	}
	nodes, err := convertOSNodesToV1Beta1(src.ESInstallArgs, src.Nodes)
	if err != nil {
		return nil, err
	}
	return &v1beta1.OpenSearchComponent{
		Enabled:  src.Enabled,
		Policies: src.Policies,
		Nodes:    nodes,
	}, nil
}

func convertOSNodesToV1Beta1(args []InstallArgs, nodes []OpenSearchNode) ([]v1beta1.OpenSearchNode, error) {
	var out []v1beta1.OpenSearchNode
	installArgNodes, err := convertInstallArgsToOSNodes(args)
	if err != nil {
		return nil, err
	}
	for _, inNode := range nodes {
		var storage *v1beta1.OpenSearchNodeStorage
		if inNode.Storage != nil {
			storage = &v1beta1.OpenSearchNodeStorage{
				Size: inNode.Storage.Size,
			}
		}
		dst := v1beta1.OpenSearchNode{
			Name:      inNode.Name,
			Replicas:  inNode.Replicas,
			Roles:     inNode.Roles,
			Storage:   storage,
			Resources: inNode.Resources,
		}

		// Merge any overlapping install arg nodes with user-supplied nodes
		if src, ok := installArgNodes[dst.Name]; ok {
			mergeOpenSearchNodes(&src, &dst)
			delete(installArgNodes, src.Name)
		}
		out = append(out, dst)
	}

	for _, node := range installArgNodes {
		out = append(out, node)
	}

	return out, nil
}

func mergeOpenSearchNodes(src, dst *v1beta1.OpenSearchNode) {
	if src.Roles != nil {
		dst.Roles = src.Roles
	}
	if src.Storage != nil {
		dst.Storage = src.Storage
	}
	if src.Replicas > 0 {
		dst.Replicas = src.Replicas
	}
	if src.Resources != nil {
		dst.Resources = src.Resources
	}
}

func convertInstallArgsToOSNodes(args []InstallArgs) (map[string]v1beta1.OpenSearchNode, error) {
	masterNode := &v1beta1.OpenSearchNode{
		Name:  masterNodeName,
		Roles: []vmov1.NodeRole{vmov1.MasterRole},
	}
	dataNode := &v1beta1.OpenSearchNode{
		Name:  dataNodeName,
		Roles: []vmov1.NodeRole{vmov1.DataRole},
	}
	ingestNode := &v1beta1.OpenSearchNode{
		Name:  ingestNodeName,
		Roles: []vmov1.NodeRole{vmov1.IngestRole},
	}
	// Helper function set the value of an int from a string
	// used to set the replica count of a node from an install arg
	setIntValue := func(val *int32, a InstallArgs) error {
		var intVal int32
		_, err := fmt.Sscan(a.Value, &intVal)
		if err != nil {
			return err
		}
		*val = intVal
		return nil
	}
	// Helper function to set the memory quantity of a node's resource requirements
	setMemory := func(node *v1beta1.OpenSearchNode, memory string) error {
		q, err := resource.ParseQuantity(memory)
		if err != nil {
			return err
		}
		node.Resources = &corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: q,
			},
		}
		return nil
	}
	for _, arg := range args {
		switch arg.Name {
		case masterNodeReplicas:
			if err := setIntValue(&masterNode.Replicas, arg); err != nil {
				return nil, err
			}
		case masterNodeMemory:
			if err := setMemory(masterNode, arg.Value); err != nil {
				return nil, err
			}
		case masterNodeStorage:
			masterNode.Storage = &v1beta1.OpenSearchNodeStorage{
				Size: arg.Value,
			}
		case dataNodeReplicas:
			if err := setIntValue(&dataNode.Replicas, arg); err != nil {
				return nil, err
			}
		case dataNodeMemory:
			if err := setMemory(dataNode, arg.Value); err != nil {
				return nil, err
			}
		case dataNodeStorage:
			dataNode.Storage = &v1beta1.OpenSearchNodeStorage{
				Size: arg.Value,
			}
		case ingestNodeReplicas:
			if err := setIntValue(&ingestNode.Replicas, arg); err != nil {
				return nil, err
			}
		case ingestNodeMemory:
			if err := setMemory(ingestNode, arg.Value); err != nil {
				return nil, err
			}
		}
	}

	return map[string]v1beta1.OpenSearchNode{
		masterNodeName: *masterNode,
		dataNodeName:   *dataNode,
		ingestNodeName: *ingestNode,
	}, nil
}

func convertFluentdToV1Beta1(src *FluentdComponent) *v1beta1.FluentdComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.FluentdComponent{
		Enabled:           src.Enabled,
		ExtraVolumeMounts: convertVolumeMountsToV1Beta1(src.ExtraVolumeMounts),
		OpenSearchURL:     src.ElasticsearchURL,
		OpenSearchSecret:  src.ElasticsearchSecret,
		OCI:               convertOCILoggingConfigurationToV1Beta1(src.OCI),
		InstallOverrides:  convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertVolumeMountsToV1Beta1(mounts []VolumeMount) []v1beta1.VolumeMount {
	var out []v1beta1.VolumeMount
	for _, mount := range mounts {
		out = append(out, v1beta1.VolumeMount{
			Source:      mount.Source,
			Destination: mount.Destination,
			ReadOnly:    mount.ReadOnly,
		})
	}
	return out
}

func convertOCILoggingConfigurationToV1Beta1(oci *OciLoggingConfiguration) *v1beta1.OciLoggingConfiguration {
	if oci == nil {
		return nil
	}
	return &v1beta1.OciLoggingConfiguration{
		DefaultAppLogID: oci.DefaultAppLogID,
		SystemLogID:     oci.SystemLogID,
		APISecret:       oci.APISecret,
	}
}

func convertGrafanaToV1Beta1(src *GrafanaComponent) *v1beta1.GrafanaComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.GrafanaComponent{
		Enabled: src.Enabled,
	}
}

func convertIngressNGINXToV1Beta1(src *IngressNginxComponent) (*v1beta1.IngressNginxComponent, error) {
	if src == nil {
		return nil, nil
	}
	installOverrides, err := convertInstallOverridesWithArgsToV1Beta1(src.NGINXInstallArgs, src.InstallOverrides)
	if err != nil {
		return nil, err
	}
	return &v1beta1.IngressNginxComponent{
		IngressClassName: src.IngressClassName,
		Type:             v1beta1.IngressType(src.Type),
		Ports:            src.Ports,
		Enabled:          src.Enabled,
		InstallOverrides: installOverrides,
	}, nil
}

func convertIstioToV1Beta1(src *IstioComponent) (*v1beta1.IstioComponent, error) {
	if src == nil {
		return nil, nil
	}
	istioYaml, err := convertIstioComponentToYaml(src)
	if err != nil {
		return nil, err
	}
	overrides := convertInstallOverridesToV1Beta1(src.InstallOverrides)
	override, err := createValueOverride([]byte(istioYaml))
	if err != nil {
		return nil, err
	}
	overrides.ValueOverrides = append(overrides.ValueOverrides, override)
	return &v1beta1.IstioComponent{
		InstallOverrides: overrides,
		Enabled:          src.Enabled,
		InjectionEnabled: src.InjectionEnabled,
	}, nil
}

func convertJaegerOperatorToV1Beta1(src *JaegerOperatorComponent) *v1beta1.JaegerOperatorComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.JaegerOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertKialiToV1Beta1(src *KialiComponent) *v1beta1.KialiComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.KialiComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertKeycloakToV1Beta1(src *KeycloakComponent) (*v1beta1.KeycloakComponent, error) {
	if src == nil {
		return nil, nil
	}
	keycloakOverrides, err := convertInstallOverridesWithArgsToV1Beta1(src.KeycloakInstallArgs, src.InstallOverrides)
	if err != nil {
		return nil, err
	}
	mysqlOverrides, err := convertInstallOverridesWithArgsToV1Beta1(src.MySQL.MySQLInstallArgs, src.MySQL.InstallOverrides)
	if err != nil {
		return nil, err
	}
	return &v1beta1.KeycloakComponent{
		MySQL: v1beta1.MySQLComponent{
			VolumeSource:     src.MySQL.VolumeSource,
			InstallOverrides: mysqlOverrides,
		},
		Enabled:          src.Enabled,
		InstallOverrides: keycloakOverrides,
	}, nil
}

func convertOSDToV1Beta1(src *KibanaComponent) *v1beta1.OpenSearchDashboardsComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.OpenSearchDashboardsComponent{
		Enabled:  src.Enabled,
		Replicas: src.Replicas,
	}
}

func convertKubeStateMetricsToV1Beta1(src *KubeStateMetricsComponent) *v1beta1.KubeStateMetricsComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.KubeStateMetricsComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertPrometheusToV1Beta1(src *PrometheusComponent) *v1beta1.PrometheusComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.PrometheusComponent{
		Enabled: src.Enabled,
	}
}

func convertPrometheusAdapterToV1Beta1(src *PrometheusAdapterComponent) *v1beta1.PrometheusAdapterComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.PrometheusAdapterComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertPrometheusNodeExporterToV1Beta1(src *PrometheusNodeExporterComponent) *v1beta1.PrometheusNodeExporterComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.PrometheusNodeExporterComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertPrometheusOperatorToV1Beta1(src *PrometheusOperatorComponent) *v1beta1.PrometheusOperatorComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.PrometheusOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertPrometheusPushGatewayToV1Beta1(src *PrometheusPushgatewayComponent) *v1beta1.PrometheusPushgatewayComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.PrometheusPushgatewayComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertRancherToV1Beta1(src *RancherComponent) *v1beta1.RancherComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.RancherComponent{
		Enabled:             src.Enabled,
		InstallOverrides:    convertInstallOverridesToV1Beta1(src.InstallOverrides),
		KeycloakAuthEnabled: src.KeycloakAuthEnabled,
	}
}

func convertRancherBackupToV1Beta1(src *RancherBackupComponent) *v1beta1.RancherBackupComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.RancherBackupComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertWeblogicOperatorToV1Beta1(src *WebLogicOperatorComponent) *v1beta1.WebLogicOperatorComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.WebLogicOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertVeleroToV1Beta1(src *VeleroComponent) *v1beta1.VeleroComponent {
	if src == nil {
		return nil
	}
	return &v1beta1.VeleroComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesToV1Beta1(src.InstallOverrides),
	}
}

func convertVerrazzanoToV1Beta1(src *VerrazzanoComponent) (*v1beta1.VerrazzanoComponent, error) {
	if src == nil {
		return nil, nil
	}
	installOverrides, err := convertInstallOverridesWithArgsToV1Beta1(src.InstallArgs, src.InstallOverrides)
	if err != nil {
		return nil, err
	}
	return &v1beta1.VerrazzanoComponent{
		Enabled:          src.Enabled,
		InstallOverrides: installOverrides,
	}, nil
}

func convertConditionsTo(conditions []Condition) []v1beta1.Condition {
	var out []v1beta1.Condition
	for _, condition := range conditions {
		out = append(out, v1beta1.Condition{
			Type:               v1beta1.ConditionType(condition.Type),
			Status:             condition.Status,
			LastTransitionTime: condition.LastTransitionTime,
			Message:            condition.Message,
		})
	}
	return out
}

func convertComponentStatusMapTo(components ComponentStatusMap) v1beta1.ComponentStatusMap {
	if components == nil {
		return nil
	}
	componentStatusMap := v1beta1.ComponentStatusMap{}
	for component, detail := range components {
		if detail != nil {
			componentStatusMap[component] = &v1beta1.ComponentStatusDetails{
				Name:                     detail.Name,
				Conditions:               convertConditionsTo(detail.Conditions),
				State:                    v1beta1.CompStateType(detail.State),
				Version:                  detail.Version,
				LastReconciledGeneration: detail.LastReconciledGeneration,
				ReconcilingGeneration:    detail.ReconcilingGeneration,
			}
		}
	}
	return componentStatusMap
}

func convertVerrazzanoInstanceTo(instance *InstanceInfo) *v1beta1.InstanceInfo {
	if instance == nil {
		return nil
	}
	return &v1beta1.InstanceInfo{
		ConsoleURL:              instance.ConsoleURL,
		KeyCloakURL:             instance.KeyCloakURL,
		RancherURL:              instance.RancherURL,
		OpenSearchURL:           instance.ElasticURL,
		OpenSearchDashboardsURL: instance.KibanaURL,
		GrafanaURL:              instance.GrafanaURL,
		PrometheusURL:           instance.PrometheusURL,
		KialiURL:                instance.KialiURL,
		JaegerURL:               instance.JaegerURL,
	}
}

func convertSecuritySpecTo(security SecuritySpec) v1beta1.SecuritySpec {
	return v1beta1.SecuritySpec{
		AdminSubjects:   security.AdminSubjects,
		MonitorSubjects: security.MonitorSubjects,
	}
}

func convertInstallOverridesWithArgsToV1Beta1(args []InstallArgs, overrides InstallOverrides) (v1beta1.InstallOverrides, error) {
	convertedOverrides := convertInstallOverridesToV1Beta1(overrides)
	if len(args) > 0 {
		merged, err := convertInstallArgsToYaml(args)
		if err != nil {
			return v1beta1.InstallOverrides{}, err
		}
		override, err := createValueOverride([]byte(merged))
		if err != nil {
			return v1beta1.InstallOverrides{}, err
		}
		convertedOverrides.ValueOverrides = append(convertedOverrides.ValueOverrides, override)
	}
	return convertedOverrides, nil
}

func convertInstallArgsToYaml(args []InstallArgs) (string, error) {
	var yamls []string
	for _, arg := range args {
		var yamlString string
		var err error
		if len(arg.ValueList) > 0 {
			yamlString, err = vzyaml.Expand(0, false, arg.Name, arg.ValueList...)
		} else {
			yamlString, err = vzyaml.Expand(0, false, arg.Name, arg.Value)
		}
		if err != nil {
			return "", err
		}
		yamls = append(yamls, yamlString)
	}

	return vzyaml.ReplacementMerge(yamls...)
}

func convertCommonKubernetesToYaml(src CommonKubernetesSpec, replicasInfo, affinityInfo expandInfo) (string, error) {
	var yamls []string
	replicaYaml, err := vzyaml.Expand(replicasInfo.leftMargin, false, replicasInfo.key, strconv.FormatUint(uint64(src.Replicas), 10))
	if err != nil {
		return "", err
	}
	yamls = append(yamls, replicaYaml)
	if src.Affinity != nil {
		affinityBytes, err := yaml.Marshal(src.Affinity)
		if err != nil {
			return "", err
		}
		affinityYaml, err := vzyaml.Expand(affinityInfo.leftMargin, false, affinityInfo.key, string(affinityBytes))
		if err != nil {
			return "", err
		}
		yamls = append(yamls, affinityYaml)
	}
	return vzyaml.ReplacementMerge(yamls...)
}

func convertInstallOverridesToV1Beta1(src InstallOverrides) v1beta1.InstallOverrides {
	return v1beta1.InstallOverrides{
		MonitorChanges: src.MonitorChanges,
		ValueOverrides: convertValueOverridesToV1Beta1(src.ValueOverrides),
	}
}

func convertValueOverridesToV1Beta1(overrides []Overrides) []v1beta1.Overrides {
	var out []v1beta1.Overrides
	for _, override := range overrides {
		out = append(out, v1beta1.Overrides{
			ConfigMapRef: override.ConfigMapRef,
			SecretRef:    override.SecretRef,
			Values:       override.Values,
		})
	}
	return out
}

func createValueOverride(rawYAML []byte) (v1beta1.Overrides, error) {
	rawJSON, err := yaml.YAMLToJSON(rawYAML)
	if err != nil {
		return v1beta1.Overrides{}, err
	}
	return v1beta1.Overrides{
		Values: &apiextensionsv1.JSON{
			Raw: rawJSON,
		},
	}, nil
}
