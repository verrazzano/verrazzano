// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const (
	testUtilDir = "./test_utils/"
	utilDir = "./utils/"
)

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			CertManager: &vzapi.CertManagerComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

var fakeComponent = certManagerComponent{}

func getBoolPtr(b bool) *bool {
	return &b
}