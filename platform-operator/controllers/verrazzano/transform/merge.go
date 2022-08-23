// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/yaml"

	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const (
	// implicit base profile (defaults)
	baseProfile = "base"
)

// MergeProfiles merges a list of v1alpha1.Verrazzano profile files with an existing Verrazzano CR.
// The profiles must be in the Verrazzano CR format
func MergeProfiles(actualCR *vzapi.Verrazzano, profileFiles ...string) (*vzapi.Verrazzano, error) {
	// First merge the profiles
	profileStrings, err := appendProfileComponentOverrides(profileFiles...)
	if err != nil {
		return nil, err
	}
	merged, err := vzyaml.StrategicMerge(vzapi.Verrazzano{}, profileStrings...)
	if err != nil {
		return nil, err
	}

	profileVerrazzano := &vzapi.Verrazzano{}
	if err := yaml.Unmarshal([]byte(merged), profileVerrazzano); err != nil {
		return nil, err
	}
	cr := actualCR.DeepCopy()
	appendComponentOverrides(cr, profileVerrazzano)

	// Now merge the the profiles on top of the Verrazzano CR
	crYAML, err := yaml.Marshal(cr)
	if err != nil {
		return nil, err
	}

	// merge all profiles together into a single yaml
	merged, err = vzyaml.StrategicMerge(vzapi.Verrazzano{}, merged, string(crYAML))
	if err != nil {
		return nil, err
	}

	// Return a new CR
	var newCR vzapi.Verrazzano
	yaml.Unmarshal([]byte(merged), &newCR)

	return &newCR, nil
}

// MergeProfiles merges a list of v1beta1.Verrazzano profile files with an existing Verrazzano CR.
// The profiles must be in the Verrazzano CR format
func MergeProfilesForV1beta1(actualCR *v1beta1.Verrazzano, profileFiles ...string) (*v1beta1.Verrazzano, error) {
	// First merge the profiles
	profileStrings, err := appendProfileComponentOverrides(profileFiles...)
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
	appendComponentOverridesV1beta1(cr, profileVerrazzano)

	// Now merge the the profiles on top of the Verrazzano CR
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
	yaml.Unmarshal([]byte(merged), &newCR)

	return &newCR, nil
}

func appendProfileComponentOverrides(profileFiles ...string) ([]string, error) {
	var profileCR *vzapi.Verrazzano
	var profileStrings []string
	for i := range profileFiles {
		profileFile := profileFiles[len(profileFiles)-1-i]
		data, err := os.ReadFile(profileFile)
		if err != nil {
			return nil, err
		}
		cr := &vzapi.Verrazzano{}
		if err := yaml.Unmarshal(data, cr); err != nil {
			return nil, err
		}
		if profileCR == nil {
			profileCR = cr
		} else {
			appendComponentOverrides(profileCR, cr)
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

// GetEffectiveCR Creates an "effective" v1alpha1.Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - last definition wins
func GetEffectiveCR(actualCR *vzapi.Verrazzano) (*vzapi.Verrazzano, error) {
	if actualCR == nil {
		return nil, nil
	}
	// Identify the set of profiles, base + declared
	profiles := []string{baseProfile, string(vzapi.Prod)}
	if len(actualCR.Spec.Profile) > 0 {
		profiles = append([]string{baseProfile}, strings.Split(string(actualCR.Spec.Profile), ",")...)
	}
	var profileFiles []string
	for _, profile := range profiles {
		profileFiles = append(profileFiles, config.GetProfile(profile))
	}
	// Merge the profile files into an effective profile YAML string
	effectiveCR, err := MergeProfiles(actualCR, profileFiles...)
	if err != nil {
		return nil, err
	}
	effectiveCR.Status = vzapi.VerrazzanoStatus{} // Don't replicate the CR status in the effective config
	// if Certificate in CertManager is empty, set it to default CA
	var emptyCertConfig = vzapi.Certificate{}
	if effectiveCR.Spec.Components.CertManager.Certificate == emptyCertConfig {
		effectiveCR.Spec.Components.CertManager.Certificate.CA = vzapi.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		}
	}
	return effectiveCR, nil
}

// GetEffectiveV1beta1CR Creates an "effective" v1beta1.Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - last definition wins
func GetEffectiveV1beta1CR(actualCR *v1beta1.Verrazzano) (*v1beta1.Verrazzano, error) {
	if actualCR == nil {
		return nil, nil
	}
	// Identify the set of profiles, base + declared
	profiles := []string{baseProfile, string(vzapi.Prod)}
	if len(actualCR.Spec.Profile) > 0 {
		profiles = append([]string{baseProfile}, strings.Split(string(actualCR.Spec.Profile), ",")...)
	}
	var profileFiles []string
	for _, profile := range profiles {
		profileFiles = append(profileFiles, config.GetProfile(profile))
	}
	// Merge the profile files into an effective profile YAML string
	effectiveCR, err := MergeProfilesForV1beta1(actualCR, profileFiles...)
	if err != nil {
		return nil, err
	}
	effectiveCR.Status = v1beta1.VerrazzanoStatus{} // Don't replicate the CR status in the effective config
	// if Certificate in CertManager is empty, set it to default CA
	var emptyCertConfig = v1beta1.Certificate{}
	if effectiveCR.Spec.Components.CertManager.Certificate == emptyCertConfig {
		effectiveCR.Spec.Components.CertManager.Certificate.CA = v1beta1.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		}
	}
	return effectiveCR, nil
}

//appendComponentOverrides copies the profile overrides of v1alpha1.Verrazzano over to the actual overrides. Any component that has overrides should be included here.
// Because overrides lacks a proper merge key, a strategic merge will replace the array instead of merging it. This function stops that replacement from occurring.
// The profile CR overrides must be appended to the actual CR overrides to preserve the precedence order in the way HelmComponent consumes them.
func appendComponentOverrides(actual, profile *vzapi.Verrazzano) {
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

//appendComponentOverridesV1beta1 copies the profile overrides of v1beta1.Verrazzano over to the actual overrides. Any component that has overrides should be included here.
// Because overrides lacks a proper merge key, a strategic merge will replace the array instead of merging it. This function stops that replacement from occurring.
// The profile CR overrides must be appended to the actual CR overrides to preserve the precedence order in the way HelmComponent consumes them.
func appendComponentOverridesV1beta1(actual, profile *v1beta1.Verrazzano) {
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

	actualIngress := actual.Spec.Components.Ingress
	profileIngress := profile.Spec.Components.Ingress
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

////mergeOverrides merges the various profiles overrides of v1beta1.Verrazzano into the actual overrides
func mergeOverrides(actual, profile []vzapi.Overrides) []vzapi.Overrides {
	return append(actual, profile...)
}

//mergeOverridesV1beta1 merges the various profiles overrides of v1beta1.Verrazzano into the actual overrides
func mergeOverridesV1beta1(actual, profile []v1beta1.Overrides) []v1beta1.Overrides {
	return append(actual, profile...)
}
