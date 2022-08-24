// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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

func (in *Verrazzano) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.Verrazzano)
	if src == nil {
		return nil
	}
	in.ObjectMeta = src.ObjectMeta

	// Convert Spec
	components, err := convertComponentsFrom(src.Spec.Components)
	if err != nil {
		return err
	}
	in.Spec.Components = components
	in.Spec.Profile = ProfileType(src.Spec.Profile)
	in.Spec.EnvironmentName = src.Spec.EnvironmentName
	in.Spec.Version = src.Spec.Version
	in.Spec.DefaultVolumeSource = src.Spec.DefaultVolumeSource
	in.Spec.Security = convertSecuritySpecFrom(src.Spec.Security)

	// Convert status
	in.Status.State = VzStateType(src.Status.State)
	in.Status.Version = src.Status.Version
	in.Status.Conditions = convertConditionsFrom(src.Status.Conditions)
	in.Status.Components = convertComponentStatusMapFrom(src.Status.Components)
	in.Status.VerrazzanoInstance = convertVerrazzanoInstanceFrom(src.Status.VerrazzanoInstance)
	return nil
}

func convertSecuritySpecFrom(security v1alpha1.SecuritySpec) SecuritySpec {
	return SecuritySpec{
		AdminSubjects:   security.AdminSubjects,
		MonitorSubjects: security.MonitorSubjects,
	}
}

func convertComponentsFrom(src v1alpha1.ComponentSpec) (ComponentSpec, error) {
	authProxyComponent, err := convertAuthProxyFrom(src.AuthProxy)
	if err != nil {
		return ComponentSpec{}, err
	}
	opensearchComponent, err := convertOpenSearchFrom(src.Elasticsearch)
	if err != nil {
		return ComponentSpec{}, err
	}
	ingressComponent, err := convertIngressNGINXFrom(src.Ingress)
	if err != nil {
		return ComponentSpec{}, err
	}
	istioComponent, err := convertIstioFrom(src.Istio)
	if err != nil {
		return ComponentSpec{}, err
	}
	keycloakComponent, err := convertKeycloakFrom(src.Keycloak)
	if err != nil {
		return ComponentSpec{}, err
	}
	verrazzanoComponent, err := convertVerrazzanoFrom(src.Verrazzano)
	if err != nil {
		return ComponentSpec{}, err
	}
	return ComponentSpec{
		CertManager:            convertCertManagerFrom(src.CertManager),
		CoherenceOperator:      convertCoherenceOperatorFrom(src.CoherenceOperator),
		ApplicationOperator:    convertApplicationOperatorFrom(src.ApplicationOperator),
		AuthProxy:              authProxyComponent,
		OAM:                    convertOAMFrom(src.OAM),
		Console:                convertConsoleFrom(src.Console),
		DNS:                    convertDNSFrom(src.DNS),
		OpenSearch:             opensearchComponent,
		Fluentd:                convertFluentdFrom(src.Fluentd),
		Grafana:                convertGrafanaFrom(src.Grafana),
		Ingress:                ingressComponent,
		Istio:                  istioComponent,
		JaegerOperator:         convertJaegerOperatorFrom(src.JaegerOperator),
		Kiali:                  convertKialiFrom(src.Kiali),
		Keycloak:               keycloakComponent,
		MySQLOperator:          convertMySQLOperatorFrom(src.MySQLOperator),
		OpenSearchDashboards:   convertOSDFrom(src.Kibana),
		KubeStateMetrics:       convertKubeStateMetricsFrom(src.KubeStateMetrics),
		Prometheus:             convertPrometheusFrom(src.Prometheus),
		PrometheusAdapter:      convertPrometheusAdapterFrom(src.PrometheusAdapter),
		PrometheusNodeExporter: convertPrometheusNodeExporterFrom(src.PrometheusNodeExporter),
		PrometheusOperator:     convertPrometheusOperatorFrom(src.PrometheusOperator),
		PrometheusPushgateway:  convertPrometheusPushGatewayFrom(src.PrometheusPushgateway),
		Rancher:                convertRancherFrom(src.Rancher),
		RancherBackup:          convertRancherBackupFrom(src.RancherBackup),
		WebLogicOperator:       convertWeblogicOperatorFrom(src.WebLogicOperator),
		Velero:                 convertVeleroFrom(src.Velero),
		Verrazzano:             verrazzanoComponent,
	}, nil
}

func convertCertManagerFrom(src *v1alpha1.CertManagerComponent) *CertManagerComponent {
	if src == nil {
		return nil
	}
	return &CertManagerComponent{
		Certificate:      convertCertificateFrom(src.Certificate),
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertCertificateFrom(certificate v1alpha1.Certificate) Certificate {
	return Certificate{
		Acme: Acme{
			Provider:     ProviderType(certificate.Acme.Provider),
			EmailAddress: certificate.Acme.EmailAddress,
			Environment:  certificate.Acme.Environment,
		},
		CA: CA{
			SecretName:               certificate.CA.SecretName,
			ClusterResourceNamespace: certificate.CA.ClusterResourceNamespace,
		},
	}
}

func convertCoherenceOperatorFrom(src *v1alpha1.CoherenceOperatorComponent) *CoherenceOperatorComponent {
	if src == nil {
		return nil
	}
	return &CoherenceOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertApplicationOperatorFrom(src *v1alpha1.ApplicationOperatorComponent) *ApplicationOperatorComponent {
	if src == nil {
		return nil
	}
	return &ApplicationOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertAuthProxyFrom(src *v1alpha1.AuthProxyComponent) (*AuthProxyComponent, error) {
	if src == nil {
		return nil, nil
	}
	overrides := convertInstallOverridesFrom(src.InstallOverrides)
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

	return &AuthProxyComponent{
		Enabled:          src.Enabled,
		InstallOverrides: overrides,
	}, nil
}

func convertOAMFrom(src *v1alpha1.OAMComponent) *OAMComponent {
	if src == nil {
		return nil
	}
	return &OAMComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertConsoleFrom(src *v1alpha1.ConsoleComponent) *ConsoleComponent {
	if src == nil {
		return nil
	}
	return &ConsoleComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertDNSFrom(src *v1alpha1.DNSComponent) *DNSComponent {
	if src == nil {
		return nil
	}
	return &DNSComponent{
		Wildcard:         convertWildcardDNSFrom(src.Wildcard),
		OCI:              convertOCIDNSFrom(src.OCI),
		External:         convertExternalDNSFrom(src.External),
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertWildcardDNSFrom(wildcard *v1alpha1.Wildcard) *Wildcard {
	if wildcard == nil {
		return nil
	}
	return &Wildcard{
		Domain: wildcard.Domain,
	}
}

func convertOCIDNSFrom(oci *v1alpha1.OCI) *OCI {
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

func convertExternalDNSFrom(external *v1alpha1.External) *External {
	if external == nil {
		return nil
	}
	return &External{Suffix: external.Suffix}
}

func convertOpenSearchFrom(src *v1alpha1.ElasticsearchComponent) (*OpenSearchComponent, error) {
	if src == nil {
		return nil, nil
	}
	nodes, err := convertOSNodesFrom(src.ESInstallArgs, src.Nodes)
	if err != nil {
		return nil, err
	}
	return &OpenSearchComponent{
		Enabled:  src.Enabled,
		Policies: src.Policies,
		Nodes:    nodes,
	}, nil
}

func convertOSNodesFrom(args []v1alpha1.InstallArgs, nodes []v1alpha1.OpenSearchNode) ([]OpenSearchNode, error) {
	out, err := convertInstallArgsToOSNodes(args)
	if err != nil {
		return nil, err
	}
	for _, inNode := range nodes {
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
		})
	}
	return out, nil
}

func convertInstallArgsToOSNodes(args []v1alpha1.InstallArgs) ([]OpenSearchNode, error) {
	masterNode := &OpenSearchNode{
		Name:  masterNodeName,
		Roles: []vmov1.NodeRole{vmov1.MasterRole},
	}
	dataNode := &OpenSearchNode{
		Name:  dataNodeName,
		Roles: []vmov1.NodeRole{vmov1.DataRole},
	}
	ingestNode := &OpenSearchNode{
		Name:  ingestNodeName,
		Roles: []vmov1.NodeRole{vmov1.IngestRole},
	}
	// Helper function set the value of an int from a string
	// used to set the replica count of a node from an install arg
	setIntValue := func(val *int32, a v1alpha1.InstallArgs) error {
		var intVal int32
		_, err := fmt.Sscan(a.Value, &intVal)
		if err != nil {
			return err
		}
		*val = intVal
		return nil
	}
	// Helper function to set the memory quantity of a node's resource requirements
	setMemory := func(node *OpenSearchNode, memory string) error {
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
			masterNode.Storage = &OpenSearchNodeStorage{
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
			dataNode.Storage = &OpenSearchNodeStorage{
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

	return []OpenSearchNode{
		*masterNode,
		*dataNode,
		*ingestNode,
	}, nil
}

func convertFluentdFrom(src *v1alpha1.FluentdComponent) *FluentdComponent {
	if src == nil {
		return nil
	}
	return &FluentdComponent{
		Enabled:           src.Enabled,
		ExtraVolumeMounts: convertVolumeMountsFrom(src.ExtraVolumeMounts),
		OpenSearchURL:     src.ElasticsearchURL,
		OpenSearchSecret:  src.ElasticsearchSecret,
		OCI:               convertOCILoggingConfigurationFrom(src.OCI),
		InstallOverrides:  convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertVolumeMountsFrom(mounts []v1alpha1.VolumeMount) []VolumeMount {
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

func convertOCILoggingConfigurationFrom(oci *v1alpha1.OciLoggingConfiguration) *OciLoggingConfiguration {
	if oci == nil {
		return nil
	}
	return &OciLoggingConfiguration{
		DefaultAppLogID: oci.DefaultAppLogID,
		SystemLogID:     oci.SystemLogID,
		APISecret:       oci.APISecret,
	}
}

func convertGrafanaFrom(src *v1alpha1.GrafanaComponent) *GrafanaComponent {
	if src == nil {
		return nil
	}
	return &GrafanaComponent{
		Enabled:  src.Enabled,
		Replicas: src.Replicas,
		Database: &DatabaseInfo{
			Host: src.Database.Host,
			Name: src.Database.Name,
		},
	}
}

func convertIngressNGINXFrom(src *v1alpha1.IngressNginxComponent) (*IngressNginxComponent, error) {
	if src == nil {
		return nil, nil
	}
	installOverrides, err := convertInstallOverridesWithArgsFrom(src.NGINXInstallArgs, src.InstallOverrides)
	if err != nil {
		return nil, err
	}
	return &IngressNginxComponent{
		IngressClassName: src.IngressClassName,
		Type:             IngressType(src.Type),
		Ports:            src.Ports,
		Enabled:          src.Enabled,
		InstallOverrides: installOverrides,
	}, nil
}

func convertIstioFrom(src *v1alpha1.IstioComponent) (*IstioComponent, error) {
	if src == nil {
		return nil, nil
	}
	istioYaml, err := convertIstioComponentToYaml(src)
	if err != nil {
		return nil, err
	}
	overrides := convertInstallOverridesFrom(src.InstallOverrides)
	override, err := createValueOverride([]byte(istioYaml))
	if err != nil {
		return nil, err
	}
	overrides.ValueOverrides = append(overrides.ValueOverrides, override)
	return &IstioComponent{
		InstallOverrides: overrides,
		Enabled:          src.Enabled,
		InjectionEnabled: src.InjectionEnabled,
	}, nil
}

func convertJaegerOperatorFrom(src *v1alpha1.JaegerOperatorComponent) *JaegerOperatorComponent {
	if src == nil {
		return nil
	}
	return &JaegerOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertKialiFrom(src *v1alpha1.KialiComponent) *KialiComponent {
	if src == nil {
		return nil
	}
	return &KialiComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertKeycloakFrom(src *v1alpha1.KeycloakComponent) (*KeycloakComponent, error) {
	if src == nil {
		return nil, nil
	}
	keycloakOverrides, err := convertInstallOverridesWithArgsFrom(src.KeycloakInstallArgs, src.InstallOverrides)
	if err != nil {
		return nil, err
	}
	mysqlOverrides, err := convertInstallOverridesWithArgsFrom(src.MySQL.MySQLInstallArgs, src.MySQL.InstallOverrides)
	if err != nil {
		return nil, err
	}
	return &KeycloakComponent{
		MySQL: MySQLComponent{
			VolumeSource:     src.MySQL.VolumeSource,
			InstallOverrides: mysqlOverrides,
		},
		Enabled:          src.Enabled,
		InstallOverrides: keycloakOverrides,
	}, nil
}

func convertMySQLOperatorFrom(src *v1alpha1.MySQLOperatorComponent) *MySQLOperatorComponent {
	if src == nil {
		return nil
	}
	return &MySQLOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertOSDFrom(src *v1alpha1.KibanaComponent) *OpenSearchDashboardsComponent {
	if src == nil {
		return nil
	}
	return &OpenSearchDashboardsComponent{
		Enabled:  src.Enabled,
		Replicas: src.Replicas,
	}
}

func convertKubeStateMetricsFrom(src *v1alpha1.KubeStateMetricsComponent) *KubeStateMetricsComponent {
	if src == nil {
		return nil
	}
	return &KubeStateMetricsComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertPrometheusFrom(src *v1alpha1.PrometheusComponent) *PrometheusComponent {
	if src == nil {
		return nil
	}
	return &PrometheusComponent{
		Enabled: src.Enabled,
	}
}

func convertPrometheusAdapterFrom(src *v1alpha1.PrometheusAdapterComponent) *PrometheusAdapterComponent {
	if src == nil {
		return nil
	}
	return &PrometheusAdapterComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertPrometheusNodeExporterFrom(src *v1alpha1.PrometheusNodeExporterComponent) *PrometheusNodeExporterComponent {
	if src == nil {
		return nil
	}
	return &PrometheusNodeExporterComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertPrometheusOperatorFrom(src *v1alpha1.PrometheusOperatorComponent) *PrometheusOperatorComponent {
	if src == nil {
		return nil
	}
	return &PrometheusOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertPrometheusPushGatewayFrom(src *v1alpha1.PrometheusPushgatewayComponent) *PrometheusPushgatewayComponent {
	if src == nil {
		return nil
	}
	return &PrometheusPushgatewayComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertRancherFrom(src *v1alpha1.RancherComponent) *RancherComponent {
	if src == nil {
		return nil
	}
	return &RancherComponent{
		Enabled:             src.Enabled,
		InstallOverrides:    convertInstallOverridesFrom(src.InstallOverrides),
		KeycloakAuthEnabled: src.KeycloakAuthEnabled,
	}
}

func convertRancherBackupFrom(src *v1alpha1.RancherBackupComponent) *RancherBackupComponent {
	if src == nil {
		return nil
	}
	return &RancherBackupComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertWeblogicOperatorFrom(src *v1alpha1.WebLogicOperatorComponent) *WebLogicOperatorComponent {
	if src == nil {
		return nil
	}
	return &WebLogicOperatorComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertVeleroFrom(src *v1alpha1.VeleroComponent) *VeleroComponent {
	if src == nil {
		return nil
	}
	return &VeleroComponent{
		Enabled:          src.Enabled,
		InstallOverrides: convertInstallOverridesFrom(src.InstallOverrides),
	}
}

func convertVerrazzanoFrom(src *v1alpha1.VerrazzanoComponent) (*VerrazzanoComponent, error) {
	if src == nil {
		return nil, nil
	}
	installOverrides, err := convertInstallOverridesWithArgsFrom(src.InstallArgs, src.InstallOverrides)
	if err != nil {
		return nil, err
	}
	return &VerrazzanoComponent{
		Enabled:          src.Enabled,
		InstallOverrides: installOverrides,
	}, nil
}

func convertConditionsFrom(conditions []v1alpha1.Condition) []Condition {
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

func convertComponentStatusMapFrom(components v1alpha1.ComponentStatusMap) ComponentStatusMap {
	if components == nil {
		return nil
	}
	componentStatusMap := ComponentStatusMap{}
	for component, detail := range components {
		if detail != nil {
			componentStatusMap[component] = &ComponentStatusDetails{
				Name:                     detail.Name,
				Conditions:               convertConditionsFrom(detail.Conditions),
				State:                    CompStateType(detail.State),
				Version:                  detail.Version,
				LastReconciledGeneration: detail.LastReconciledGeneration,
				ReconcilingGeneration:    detail.ReconcilingGeneration,
			}
		}
	}
	return componentStatusMap
}

func convertVerrazzanoInstanceFrom(instance *v1alpha1.InstanceInfo) *InstanceInfo {
	if instance == nil {
		return nil
	}
	return &InstanceInfo{
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

func convertInstallOverridesWithArgsFrom(args []v1alpha1.InstallArgs, overrides v1alpha1.InstallOverrides) (InstallOverrides, error) {
	convertedOverrides := convertInstallOverridesFrom(overrides)
	if len(args) > 0 {
		merged, err := convertInstallArgsToYaml(args)
		if err != nil {
			return InstallOverrides{}, err
		}
		override, err := createValueOverride([]byte(merged))
		if err != nil {
			return InstallOverrides{}, err
		}
		convertedOverrides.ValueOverrides = append(convertedOverrides.ValueOverrides, override)
	}
	return convertedOverrides, nil
}

func convertInstallArgsToYaml(args []v1alpha1.InstallArgs) (string, error) {
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

func convertCommonKubernetesToYaml(src v1alpha1.CommonKubernetesSpec, replicasInfo, affinityInfo expandInfo) (string, error) {
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

func convertInstallOverridesFrom(src v1alpha1.InstallOverrides) InstallOverrides {
	return InstallOverrides{
		MonitorChanges: src.MonitorChanges,
		ValueOverrides: convertValueOverridesFrom(src.ValueOverrides),
	}
}

func convertValueOverridesFrom(overrides []v1alpha1.Overrides) []Overrides {
	var out []Overrides
	for _, override := range overrides {
		out = append(out, Overrides{
			ConfigMapRef: override.ConfigMapRef,
			SecretRef:    override.SecretRef,
			Values:       override.Values,
		})
	}
	return out
}

func createValueOverride(rawYAML []byte) (Overrides, error) {
	rawJSON, err := yaml.YAMLToJSON(rawYAML)
	if err != nil {
		return Overrides{}, err
	}
	return Overrides{
		Values: &apiextensionsv1.JSON{
			Raw: rawJSON,
		},
	}, nil
}
