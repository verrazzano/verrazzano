// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// source file: application-operator/controllers/metricstrait/operator.go
package controllers

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"regexp"
)

func FetchTraitDefaults(workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, bool, error) {
	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil {
		print(err)
	}

	workloadType := GetSupportedWorkloadType(apiVerKind)
	switch workloadType {
	case constants.WorkloadTypeWeblogic:
		spec, err := NewTraitDefaultsForWLSDomainWorkload(workload)
		return spec, true, err
	case constants.WorkloadTypeCoherence:
		spec, err := NewTraitDefaultsForCOHWorkload(workload)
		return spec, true, err
	case constants.WorkloadTypeGeneric:
		spec, err := NewTraitDefaultsForGenericWorkload()
		return spec, true, err
	default:
		// Log the kind/workload is unsupported and return a nil trait.
		return nil, false, nil
	}

}

func GetSupportedWorkloadType(apiVerKind string) string {
	// Match any version of Group=weblogic.oracle and Kind=Domain
	if matched, _ := regexp.MatchString("^weblogic.oracle/.*\\.Domain$", apiVerKind); matched {
		return constants.WorkloadTypeWeblogic
	}
	// Match any version of Group=coherence.oracle and Kind=Coherence
	if matched, _ := regexp.MatchString("^coherence.oracle.com/.*\\.Coherence$", apiVerKind); matched {
		return constants.WorkloadTypeCoherence
	}

	// Match any version of Group=coherence.oracle and Kind=VerrazzanoHelidonWorkload or
	// In the case of Helidon, the workload isn't currently being unwrapped
	if matched, _ := regexp.MatchString("^oam.verrazzano.io/.*\\.VerrazzanoHelidonWorkload$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	// Match any version of Group=core.oam.dev and Kind=ContainerizedWorkload
	if matched, _ := regexp.MatchString("^core.oam.dev/.*\\.ContainerizedWorkload$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	// Match any version of Group=apps and Kind=Deployment
	if matched, _ := regexp.MatchString("^apps/.*\\.Deployment$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	return ""
}