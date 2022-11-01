// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package profiles

import (
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"os"
	"sigs.k8s.io/yaml"
)

// MergeProfiles merges a list of v1alpha1.Verrazzano profile files with an existing Verrazzano CR.
// The profiles must be in the Verrazzano CR format
func MergeProfiles(actualCR *v1alpha1.Verrazzano, profileFiles ...string) (*v1alpha1.Verrazzano, error) {
	// First merge the profiles
	profileStrings, err := appendProfileComponentOverrides(profileFiles...)
	if err != nil {
		return nil, err
	}
	merged, err := vzyaml.StrategicMerge(v1alpha1.Verrazzano{}, profileStrings...)
	if err != nil {
		return nil, err
	}

	profileVerrazzano := &v1alpha1.Verrazzano{}
	if err := yaml.Unmarshal([]byte(merged), profileVerrazzano); err != nil {
		return nil, err
	}
	cr := actualCR.DeepCopy()
	AppendComponentOverrides(cr, profileVerrazzano)

	// Now merge the profiles on top of the Verrazzano CR
	crYAML, err := yaml.Marshal(cr)
	if err != nil {
		return nil, err
	}

	// merge all profiles together into a single yaml
	merged, err = vzyaml.StrategicMerge(v1alpha1.Verrazzano{}, merged, string(crYAML))
	if err != nil {
		return nil, err
	}

	// Return a new CR
	var newCR v1alpha1.Verrazzano
	err = yaml.Unmarshal([]byte(merged), &newCR)
	if err != nil {
		return nil, err
	}

	mergeOSNodesV1alpha1(actualCR, &newCR)
	return &newCR, nil
}

// MergeProfilesForV1beta1 merges a list of v1beta1.Verrazzano profile files with an existing Verrazzano CR.
// The profiles must be in the Verrazzano CR format
func MergeProfilesForV1beta1(actualCR *v1beta1.Verrazzano, profileFiles ...string) (*v1beta1.Verrazzano, error) {
	// First merge the profiles
	profileStrings, err := appendProfileComponentOverridesV1beta1(profileFiles...)
	if err != nil {
		return nil, err
	}
	merged, err := vzyaml.StrategicMerge(v1beta1.Verrazzano{}, profileStrings...)
	if err != nil {
		return nil, err
	}

	profileVerrazzano := &v1beta1.Verrazzano{}
	if err := yaml.Unmarshal([]byte(merged), profileVerrazzano); err != nil {
		return nil, err
	}
	cr := actualCR.DeepCopy()
	AppendComponentOverridesV1beta1(cr, profileVerrazzano)

	// Now merge the profiles on top of the Verrazzano CR
	crYAML, err := yaml.Marshal(cr)
	if err != nil {
		return nil, err
	}

	// merge all profiles together into a single yaml
	merged, err = vzyaml.StrategicMerge(v1beta1.Verrazzano{}, merged, string(crYAML))
	if err != nil {
		return nil, err
	}

	// Return a new CR
	var newCR v1beta1.Verrazzano
	err = yaml.Unmarshal([]byte(merged), &newCR)
	if err != nil {
		return nil, err
	}
	mergeOSNodesV1beta1(actualCR, &newCR)
	return &newCR, nil
}

func appendProfileComponentOverrides(profileFiles ...string) ([]string, error) {
	var profileCR *v1alpha1.Verrazzano
	var profileStrings []string
	for i := range profileFiles {
		profileFile := profileFiles[len(profileFiles)-1-i]
		data, err := os.ReadFile(profileFile)
		if err != nil {
			return nil, err
		}
		cr := &v1alpha1.Verrazzano{}
		if err := yaml.Unmarshal(data, cr); err != nil {
			return nil, err
		}
		if profileCR == nil {
			profileCR = cr
		} else {
			AppendComponentOverrides(profileCR, cr)
			profileStrings = append(profileStrings, string(data))
		}

	}
	data, err := yaml.Marshal(profileCR)
	if err != nil {
		return nil, err
	}
	profileStrings = append(profileStrings, string(data))
	return profileStrings, nil
}

func appendProfileComponentOverridesV1beta1(profileFiles ...string) ([]string, error) {
	var profileCR *v1beta1.Verrazzano
	var profileStrings []string
	for i := range profileFiles {
		profileFile := profileFiles[len(profileFiles)-1-i]
		data, err := os.ReadFile(profileFile)
		if err != nil {
			return nil, err
		}
		cr := &v1beta1.Verrazzano{}
		if err := yaml.Unmarshal(data, cr); err != nil {
			return nil, err
		}
		if profileCR == nil {
			profileCR = cr
		} else {
			AppendComponentOverridesV1beta1(profileCR, cr)
			profileStrings = append(profileStrings, string(data))
		}

	}
	data, err := yaml.Marshal(profileCR)
	if err != nil {
		return nil, err
	}
	profileStrings = append(profileStrings, string(data))
	return profileStrings, nil
}

// AppendComponentOverrides copies the profile overrides of v1alpha1.Verrazzano over to the actual overrides. Any component that has overrides should be included here.
// Because overrides lacks a proper merge key, a strategic merge will replace the array instead of merging it. This function stops that replacement from occurring.
// The profile CR overrides must be appended to the actual CR overrides to preserve the precedence order in the way HelmComponent consumes them.
func AppendComponentOverrides(actual, profile *v1alpha1.Verrazzano) {
	actualKubeStateMetrics := actual.Spec.Components.KubeStateMetrics
	profileKubeStateMetrics := profile.Spec.Components.KubeStateMetrics
	if actualKubeStateMetrics != nil && profileKubeStateMetrics != nil {
		actualKubeStateMetrics.ValueOverrides = mergeOverrides(actualKubeStateMetrics.ValueOverrides, profileKubeStateMetrics.ValueOverrides)
	}

	actualPrometheusAdapter := actual.Spec.Components.PrometheusAdapter
	profilePrometheusAdapter := profile.Spec.Components.PrometheusAdapter
	if actualPrometheusAdapter != nil && profilePrometheusAdapter != nil {
		actualPrometheusAdapter.ValueOverrides = mergeOverrides(actualPrometheusAdapter.ValueOverrides, profilePrometheusAdapter.ValueOverrides)
	}

	actualPrometheusNodeExporter := actual.Spec.Components.PrometheusNodeExporter
	profilePrometheusNodeExporter := profile.Spec.Components.PrometheusNodeExporter
	if actualPrometheusNodeExporter != nil && profilePrometheusNodeExporter != nil {
		actualPrometheusNodeExporter.ValueOverrides = mergeOverrides(actualPrometheusNodeExporter.ValueOverrides, profilePrometheusNodeExporter.ValueOverrides)
	}

	actualPrometheusOperator := actual.Spec.Components.PrometheusOperator
	profilePrometheusOperator := profile.Spec.Components.PrometheusOperator
	if actualPrometheusOperator != nil && profilePrometheusOperator != nil {
		actualPrometheusOperator.ValueOverrides = mergeOverrides(actualPrometheusOperator.ValueOverrides, profilePrometheusOperator.ValueOverrides)
	}

	actualPrometheusPushgateway := actual.Spec.Components.PrometheusPushgateway
	profilePrometheusPushgateway := profile.Spec.Components.PrometheusPushgateway
	if actualPrometheusPushgateway != nil && profilePrometheusPushgateway != nil {
		actualPrometheusPushgateway.ValueOverrides = mergeOverrides(actualPrometheusPushgateway.ValueOverrides, profilePrometheusPushgateway.ValueOverrides)
	}

	actualCertManager := actual.Spec.Components.CertManager
	profileCertManager := profile.Spec.Components.CertManager
	if actualCertManager != nil && profileCertManager != nil {
		actualCertManager.ValueOverrides = mergeOverrides(actualCertManager.ValueOverrides, profileCertManager.ValueOverrides)
	}

	actualCoherenceOperator := actual.Spec.Components.CoherenceOperator
	profileCoherenceOperator := profile.Spec.Components.CoherenceOperator
	if actualCoherenceOperator != nil && profileCoherenceOperator != nil {
		actualCoherenceOperator.ValueOverrides = mergeOverrides(actualCoherenceOperator.ValueOverrides, profileCoherenceOperator.ValueOverrides)
	}

	actualApplicationOperator := actual.Spec.Components.ApplicationOperator
	profileApplicationOperator := profile.Spec.Components.ApplicationOperator
	if actualApplicationOperator != nil && profileApplicationOperator != nil {
		actualApplicationOperator.ValueOverrides = mergeOverrides(actualApplicationOperator.ValueOverrides, profileApplicationOperator.ValueOverrides)
	}

	actualAuthProxy := actual.Spec.Components.AuthProxy
	profileAuthProxy := profile.Spec.Components.AuthProxy
	if actualAuthProxy != nil && profileAuthProxy != nil {
		actualAuthProxy.ValueOverrides = mergeOverrides(actualAuthProxy.ValueOverrides, profileAuthProxy.ValueOverrides)
	}

	actualOAM := actual.Spec.Components.OAM
	profileOAM := profile.Spec.Components.OAM
	if actualOAM != nil && profileOAM != nil {
		actualOAM.ValueOverrides = mergeOverrides(actualOAM.ValueOverrides, profileOAM.ValueOverrides)
	}

	actualVerrazzano := actual.Spec.Components.Verrazzano
	profileVerrazzano := profile.Spec.Components.Verrazzano
	if actualVerrazzano != nil && profileVerrazzano != nil {
		actualVerrazzano.ValueOverrides = mergeOverrides(actualVerrazzano.ValueOverrides, profileVerrazzano.ValueOverrides)
	}

	actualKiali := actual.Spec.Components.Kiali
	profileKiali := profile.Spec.Components.Kiali
	if actualKiali != nil && profileKiali != nil {
		actualKiali.ValueOverrides = mergeOverrides(actualKiali.ValueOverrides, profileKiali.ValueOverrides)
	}

	actualConsole := actual.Spec.Components.Console
	profileConsole := profile.Spec.Components.Console
	if actualConsole != nil && profileConsole != nil {
		actualConsole.ValueOverrides = mergeOverrides(actualConsole.ValueOverrides, profileConsole.ValueOverrides)
	}

	actualDNS := actual.Spec.Components.DNS
	profileDNS := profile.Spec.Components.DNS
	if actualDNS != nil && profileDNS != nil {
		actualDNS.ValueOverrides = mergeOverrides(actualDNS.ValueOverrides, profileDNS.ValueOverrides)
	}

	actualIngress := actual.Spec.Components.Ingress
	profileIngress := profile.Spec.Components.Ingress
	if actualIngress != nil && profileIngress != nil {
		actualIngress.ValueOverrides = mergeOverrides(actualIngress.ValueOverrides, profileIngress.ValueOverrides)
	}

	actualIstio := actual.Spec.Components.Istio
	profileIstio := profile.Spec.Components.Istio
	if actualIstio != nil && profileIstio != nil {
		actualIstio.ValueOverrides = mergeOverrides(actualIstio.ValueOverrides, profileIstio.ValueOverrides)
	}

	actualJaegerOperator := actual.Spec.Components.JaegerOperator
	profileJaegerOperator := profile.Spec.Components.JaegerOperator
	if actualJaegerOperator != nil && profileJaegerOperator != nil {
		actualJaegerOperator.ValueOverrides = mergeOverrides(actualJaegerOperator.ValueOverrides, profileJaegerOperator.ValueOverrides)
	}

	actualKeycloak := actual.Spec.Components.Keycloak
	profileKeycloak := profile.Spec.Components.Keycloak
	if actualKeycloak != nil && profileKeycloak != nil {
		actualKeycloak.ValueOverrides = mergeOverrides(actualKeycloak.ValueOverrides, profileKeycloak.ValueOverrides)
		actualKeycloak.MySQL.ValueOverrides = mergeOverrides(actualKeycloak.MySQL.ValueOverrides, profileKeycloak.MySQL.ValueOverrides)
	}

	actualRancher := actual.Spec.Components.Rancher
	profileRancher := profile.Spec.Components.Rancher
	if actualRancher != nil && profileRancher != nil {
		actualRancher.ValueOverrides = mergeOverrides(actualRancher.ValueOverrides, profileRancher.ValueOverrides)
	}

	actualFluentd := actual.Spec.Components.Fluentd
	profileFluentd := profile.Spec.Components.Fluentd
	if actualFluentd != nil && profileFluentd != nil {
		actualFluentd.ValueOverrides = mergeOverrides(actualFluentd.ValueOverrides, profileFluentd.ValueOverrides)
	}

	actualWebLogicOperator := actual.Spec.Components.WebLogicOperator
	profileWebLogicOperator := profile.Spec.Components.WebLogicOperator
	if actualWebLogicOperator != nil && profileWebLogicOperator != nil {
		actualWebLogicOperator.ValueOverrides = mergeOverrides(actualWebLogicOperator.ValueOverrides, profileWebLogicOperator.ValueOverrides)
	}

	actualVelero := actual.Spec.Components.Velero
	profileVelero := profile.Spec.Components.Velero
	if actualVelero != nil && profileVelero != nil {
		actualVelero.ValueOverrides = mergeOverrides(actualVelero.ValueOverrides, profileVelero.ValueOverrides)
	}
}

// mergeOSNodesV1alpha1 works around omitempty replicas for default node groups
func mergeOSNodesV1alpha1(actual, merged *v1alpha1.Verrazzano) {
	actualOpensearch := actual.Spec.Components.Elasticsearch
	profileOpensearch := merged.Spec.Components.Elasticsearch
	if actualOpensearch != nil && profileOpensearch != nil {
		for i := range actualOpensearch.Nodes {
			for j := range profileOpensearch.Nodes {
				if actualOpensearch.Nodes[i].Name == profileOpensearch.Nodes[j].Name {
					profileOpensearch.Nodes[j].Replicas = actualOpensearch.Nodes[i].Replicas
				}
			}
		}
	}
}

// mergeOSNodesV1beta1 works around omitempty replicas for default node groups
func mergeOSNodesV1beta1(actual, merged *v1beta1.Verrazzano) {
	actualOpensearch := actual.Spec.Components.OpenSearch
	profileOpensearch := merged.Spec.Components.OpenSearch
	if actualOpensearch != nil && profileOpensearch != nil {
		for i := range actualOpensearch.Nodes {
			for j := range profileOpensearch.Nodes {
				if actualOpensearch.Nodes[i].Name == profileOpensearch.Nodes[j].Name {
					profileOpensearch.Nodes[j].Replicas = actualOpensearch.Nodes[i].Replicas
				}
			}
		}
	}
}

// AppendComponentOverridesV1beta1 copies the profile overrides of v1beta1.Verrazzano over to the actual overrides. Any component that has overrides should be included here.
// Because overrides lacks a proper merge key, a strategic merge will replace the array instead of merging it. This function stops that replacement from occurring.
// The profile CR overrides must be appended to the actual CR overrides to preserve the precedence order in the way HelmComponent consumes them.
func AppendComponentOverridesV1beta1(actual, profile *v1beta1.Verrazzano) {
	actualKubeStateMetrics := actual.Spec.Components.KubeStateMetrics
	profileKubeStateMetrics := profile.Spec.Components.KubeStateMetrics
	if actualKubeStateMetrics != nil && profileKubeStateMetrics != nil {
		actualKubeStateMetrics.ValueOverrides = mergeOverridesV1beta1(actualKubeStateMetrics.ValueOverrides, profileKubeStateMetrics.ValueOverrides)
	}

	actualPrometheusAdapter := actual.Spec.Components.PrometheusAdapter
	profilePrometheusAdapter := profile.Spec.Components.PrometheusAdapter
	if actualPrometheusAdapter != nil && profilePrometheusAdapter != nil {
		actualPrometheusAdapter.ValueOverrides = mergeOverridesV1beta1(actualPrometheusAdapter.ValueOverrides, profilePrometheusAdapter.ValueOverrides)
	}

	actualPrometheusNodeExporter := actual.Spec.Components.PrometheusNodeExporter
	profilePrometheusNodeExporter := profile.Spec.Components.PrometheusNodeExporter
	if actualPrometheusNodeExporter != nil && profilePrometheusNodeExporter != nil {
		actualPrometheusNodeExporter.ValueOverrides = mergeOverridesV1beta1(actualPrometheusNodeExporter.ValueOverrides, profilePrometheusNodeExporter.ValueOverrides)
	}

	actualPrometheusOperator := actual.Spec.Components.PrometheusOperator
	profilePrometheusOperator := profile.Spec.Components.PrometheusOperator
	if actualPrometheusOperator != nil && profilePrometheusOperator != nil {
		actualPrometheusOperator.ValueOverrides = mergeOverridesV1beta1(actualPrometheusOperator.ValueOverrides, profilePrometheusOperator.ValueOverrides)
	}

	actualPrometheusPushgateway := actual.Spec.Components.PrometheusPushgateway
	profilePrometheusPushgateway := profile.Spec.Components.PrometheusPushgateway
	if actualPrometheusPushgateway != nil && profilePrometheusPushgateway != nil {
		actualPrometheusPushgateway.ValueOverrides = mergeOverridesV1beta1(actualPrometheusPushgateway.ValueOverrides, profilePrometheusPushgateway.ValueOverrides)
	}

	actualCertManager := actual.Spec.Components.CertManager
	profileCertManager := profile.Spec.Components.CertManager
	if actualCertManager != nil && profileCertManager != nil {
		actualCertManager.ValueOverrides = mergeOverridesV1beta1(actualCertManager.ValueOverrides, profileCertManager.ValueOverrides)
	}

	actualCoherenceOperator := actual.Spec.Components.CoherenceOperator
	profileCoherenceOperator := profile.Spec.Components.CoherenceOperator
	if actualCoherenceOperator != nil && profileCoherenceOperator != nil {
		actualCoherenceOperator.ValueOverrides = mergeOverridesV1beta1(actualCoherenceOperator.ValueOverrides, profileCoherenceOperator.ValueOverrides)
	}

	actualApplicationOperator := actual.Spec.Components.ApplicationOperator
	profileApplicationOperator := profile.Spec.Components.ApplicationOperator
	if actualApplicationOperator != nil && profileApplicationOperator != nil {
		actualApplicationOperator.ValueOverrides = mergeOverridesV1beta1(actualApplicationOperator.ValueOverrides, profileApplicationOperator.ValueOverrides)
	}

	actualAuthProxy := actual.Spec.Components.AuthProxy
	profileAuthProxy := profile.Spec.Components.AuthProxy
	if actualAuthProxy != nil && profileAuthProxy != nil {
		actualAuthProxy.ValueOverrides = mergeOverridesV1beta1(actualAuthProxy.ValueOverrides, profileAuthProxy.ValueOverrides)
	}

	actualOAM := actual.Spec.Components.OAM
	profileOAM := profile.Spec.Components.OAM
	if actualOAM != nil && profileOAM != nil {
		actualOAM.ValueOverrides = mergeOverridesV1beta1(actualOAM.ValueOverrides, profileOAM.ValueOverrides)
	}

	actualVerrazzano := actual.Spec.Components.Verrazzano
	profileVerrazzano := profile.Spec.Components.Verrazzano
	if actualVerrazzano != nil && profileVerrazzano != nil {
		actualVerrazzano.ValueOverrides = mergeOverridesV1beta1(actualVerrazzano.ValueOverrides, profileVerrazzano.ValueOverrides)
	}

	actualKiali := actual.Spec.Components.Kiali
	profileKiali := profile.Spec.Components.Kiali
	if actualKiali != nil && profileKiali != nil {
		actualKiali.ValueOverrides = mergeOverridesV1beta1(actualKiali.ValueOverrides, profileKiali.ValueOverrides)
	}

	actualConsole := actual.Spec.Components.Console
	profileConsole := profile.Spec.Components.Console
	if actualConsole != nil && profileConsole != nil {
		actualConsole.ValueOverrides = mergeOverridesV1beta1(actualConsole.ValueOverrides, profileConsole.ValueOverrides)
	}

	actualDNS := actual.Spec.Components.DNS
	profileDNS := profile.Spec.Components.DNS
	if actualDNS != nil && profileDNS != nil {
		actualDNS.ValueOverrides = mergeOverridesV1beta1(actualDNS.ValueOverrides, profileDNS.ValueOverrides)
	}

	actualIngress := actual.Spec.Components.IngressNGINX
	profileIngress := profile.Spec.Components.IngressNGINX
	if actualIngress != nil && profileIngress != nil {
		actualIngress.ValueOverrides = mergeOverridesV1beta1(actualIngress.ValueOverrides, profileIngress.ValueOverrides)
	}

	actualIstio := actual.Spec.Components.Istio
	profileIstio := profile.Spec.Components.Istio
	if actualIstio != nil && profileIstio != nil {
		actualIstio.ValueOverrides = mergeOverridesV1beta1(actualIstio.ValueOverrides, profileIstio.ValueOverrides)
	}

	actualJaegerOperator := actual.Spec.Components.JaegerOperator
	profileJaegerOperator := profile.Spec.Components.JaegerOperator
	if actualJaegerOperator != nil && profileJaegerOperator != nil {
		actualJaegerOperator.ValueOverrides = mergeOverridesV1beta1(actualJaegerOperator.ValueOverrides, profileJaegerOperator.ValueOverrides)
	}

	actualKeycloak := actual.Spec.Components.Keycloak
	profileKeycloak := profile.Spec.Components.Keycloak
	if actualKeycloak != nil && profileKeycloak != nil {
		actualKeycloak.ValueOverrides = mergeOverridesV1beta1(actualKeycloak.ValueOverrides, profileKeycloak.ValueOverrides)
		actualKeycloak.MySQL.ValueOverrides = mergeOverridesV1beta1(actualKeycloak.MySQL.ValueOverrides, profileKeycloak.MySQL.ValueOverrides)
	}

	actualRancher := actual.Spec.Components.Rancher
	profileRancher := profile.Spec.Components.Rancher
	if actualRancher != nil && profileRancher != nil {
		actualRancher.ValueOverrides = mergeOverridesV1beta1(actualRancher.ValueOverrides, profileRancher.ValueOverrides)
	}

	actualFluentd := actual.Spec.Components.Fluentd
	profileFluentd := profile.Spec.Components.Fluentd
	if actualFluentd != nil && profileFluentd != nil {
		actualFluentd.ValueOverrides = mergeOverridesV1beta1(actualFluentd.ValueOverrides, profileFluentd.ValueOverrides)
	}

	actualWebLogicOperator := actual.Spec.Components.WebLogicOperator
	profileWebLogicOperator := profile.Spec.Components.WebLogicOperator
	if actualWebLogicOperator != nil && profileWebLogicOperator != nil {
		actualWebLogicOperator.ValueOverrides = mergeOverridesV1beta1(actualWebLogicOperator.ValueOverrides, profileWebLogicOperator.ValueOverrides)
	}

	actualVelero := actual.Spec.Components.Velero
	profileVelero := profile.Spec.Components.Velero
	if actualVelero != nil && profileVelero != nil {
		actualVelero.ValueOverrides = mergeOverridesV1beta1(actualVelero.ValueOverrides, profileVelero.ValueOverrides)
	}
}

// mergeOverrides merges the various profiles overrides of v1beta1.Verrazzano into the actual overrides
func mergeOverrides(actual, profile []v1alpha1.Overrides) []v1alpha1.Overrides {
	return append(actual, profile...)
}

// mergeOverridesV1beta1 merges the various profiles overrides of v1beta1.Verrazzano into the actual overrides
func mergeOverridesV1beta1(actual, profile []v1beta1.Overrides) []v1beta1.Overrides {
	return append(actual, profile...)
}
