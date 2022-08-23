// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"os"
	"path"
	"sigs.k8s.io/yaml"
)

const (
	testCaseBasic             = "basic"
	testCaseStatus            = "status"
	testCaseInstallArgs       = "frominstallargs"
	testCaseIstioInstallArgs  = "fromistioinstallargs"
	testCaseIstioAffinityArgs = "fromistioaffinityargs"
	testCaseFromAllComps      = "fromallcomps"
	testCaseOpensearch        = "fromopensearch"
	testCaseInstallArgsErr    = "frominstallargserr"
	testCaseToAllComps        = "toallcomps"
	testCaseRancherKeycloak   = "rancherkeycloak"
)

type converisonTestCase struct {
	name     string
	testCase string
	hasError bool
}

func loadV1Alpha1CR(testCase string) (*v1alpha1.Verrazzano, error) {
	data, err := loadTestCase(testCase, "v1alpha1")
	if err != nil {
		return nil, err
	}
	vz := &v1alpha1.Verrazzano{}
	if err := yaml.Unmarshal(data, vz); err != nil {
		return nil, err
	}
	return vz, nil
}

func loadV1Beta1(testCase string) (*Verrazzano, error) {
	data, err := loadTestCase(testCase, "v1beta1")
	if err != nil {
		return nil, err
	}
	vz := &Verrazzano{}
	if err := yaml.Unmarshal(data, vz); err != nil {
		return nil, err
	}
	return vz, nil
}

func loadTestCase(testCase, version string) ([]byte, error) {
	return os.ReadFile(path.Join("testdata", testCase, fmt.Sprintf("%s.yaml", version)))
}
