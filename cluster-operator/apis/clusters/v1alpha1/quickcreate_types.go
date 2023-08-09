// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

type (
	IdentityRef struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	Verrazzano struct {
		Install bool                    `json:"install"`
		Tag     *string                 `json:"tag"`
		Version *string                 `json:"version"`
		Spec    *v1beta1.VerrazzanoSpec `json:"spec"`
	}

	NodeConfig struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name          string  `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"version"`
		Shape         *string `json:"shape"`
		OCPUs         *int    `json:"ocpus,omitempty"`
		MemoryGbs     *int    `json:"memoryGbs,omitempty"`
		BootVolumeGbs *int    `json:"bootVolumeGbs"`
		Replicas      *int    `json:"replicas"`
	}

	Subnets struct {
		ControlPlane string `json:"controlPlane"`
		Worker       string `json:"workers"`
		LoadBalancer string `json:"loadBalancer"`
	}

	PrivateRegistry struct {
		Server string `json:"server"`
	}

	Proxy struct {
		Server string `json:"server"`
	}
)
