// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package validators

import (
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"
	"strings"
)

const (
	// Used by this test to make an illegal edit, to ensure that the apply is rejected but still generate the
	// warnings we're looking for
	illegalEnvironmentName = "asfsdafsdafadsfdsafdsafdsafadsfadsfadsfadsfdsafadsfsdafdsafasdfasdfsadfasdfasdfdasfasdfasdfasadsfasdfdasfasdfdsafdsafasd"

	podSpecJSON = `
{
  "podSpec": {
    "affinity": {
      "podAffinity": {
        "preferredDuringSchedulingIgnoredDuringExecution": [
          {
            "weight": 200,
            "podAffinityTerm": {
              "labelSelector": {
                "matchLabels": {
                  "app.kubernetes.io/instance": "mysql-innodbcluster-mysql-mysql-server",
                  "app.kubernetes.io/name": "mysql-innodbcluster-mysql-server"
                }
              },
              "topologyKey": "kubernetes.io/hostname"
            }
          }
        ]
      }
    }
  }
}`

	warningSubstring = "Modifications to MySQL server pod specs do not trigger an automatic restart of the stateful set"
)

type warningInfo struct {
	code  int
	agent string
	text  string
}

type mysqlPodSpecUpdater struct {
	warnings []warningInfo
}

func (j *mysqlPodSpecUpdater) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	// Attempt to make an illegal edit to the Jaeger configuration to ensure its component validation is working properly
	trueValue := true
	if cr.Spec.Components.JaegerOperator == nil {
		cr.Spec.Components.JaegerOperator = &v1beta1.JaegerOperatorComponent{}
	}
	// Include an illegal edit to prevent the edit from succeeding
	cr.Spec.EnvironmentName = illegalEnvironmentName
	if cr.Spec.Components.Keycloak == nil {
		cr.Spec.Components.Keycloak = &v1beta1.KeycloakComponent{}
	}
	cr.Spec.Components.Keycloak.Enabled = &trueValue
	illegalValuesObj := generateJSONValuesObj()
	cr.Spec.Components.Keycloak.MySQL.InstallOverrides.ValueOverrides = append(
		cr.Spec.Components.Keycloak.MySQL.InstallOverrides.ValueOverrides,
		v1beta1.Overrides{Values: illegalValuesObj})
}

func (j *mysqlPodSpecUpdater) ModifyCR(cr *v1alpha1.Verrazzano) {
	// Attempt to make an illegal edit to the Jaeger configuration to ensure its component validation is working properly
	trueValue := true
	if cr.Spec.Components.JaegerOperator == nil {
		cr.Spec.Components.JaegerOperator = &v1alpha1.JaegerOperatorComponent{}
	}
	// Include an illegal edit to prevent the edit from succeeding
	cr.Spec.EnvironmentName = illegalEnvironmentName
	if cr.Spec.Components.Keycloak == nil {
		cr.Spec.Components.Keycloak = &v1alpha1.KeycloakComponent{}
	}
	cr.Spec.Components.Keycloak.Enabled = &trueValue
	illegalValuesObj := generateJSONValuesObj()
	cr.Spec.Components.Keycloak.MySQL.InstallOverrides.ValueOverrides = append(
		cr.Spec.Components.Keycloak.MySQL.InstallOverrides.ValueOverrides,
		v1alpha1.Overrides{Values: illegalValuesObj})
}

func (j *mysqlPodSpecUpdater) HandleWarningHeader(code int, agent string, text string) {
	j.warnings = append(j.warnings, warningInfo{code: code, agent: agent, text: text})
}

func (j *mysqlPodSpecUpdater) hasWarnings() bool {
	return j.numWarnings() > 0
}

func (j *mysqlPodSpecUpdater) numWarnings() int {
	return len(j.warnings)
}

func (j *mysqlPodSpecUpdater) hasWarningText(substring string) bool {
	for _, warning := range j.warnings {
		if strings.Contains(warning.text, substring) {
			return true
		}
	}
	return false
}

var _ update.CRModifier = &mysqlPodSpecUpdater{}
var _ update.CRModifierV1beta1 = &mysqlPodSpecUpdater{}
var _ rest.WarningHandler = &mysqlPodSpecUpdater{}

func generateJSONValuesObj() *apiextensionsv1.JSON {
	illegalOverride := podSpecJSON
	illegalValuesObj := &apiextensionsv1.JSON{
		Raw: []byte(illegalOverride),
	}
	return illegalValuesObj
}

// runValidatorTestV1Beta1 Attempt to generate the MySQL values webhook warning, with an illegal edit so the overall edit is rejected
func runMySQLPodspecEditWarningTestV1Beta1() {
	updater := &mysqlPodSpecUpdater{}
	checkExpectations(update.UpdateCRV1beta1(updater), updater)
}

// runValidatorTestV1Alpha1 Attempt to use an illegal overrides value on the Jaeger operator configuration using the v1alpha1 API
func runMySQLPodspecEditWarningTestV1Alpha1() {
	updater := &mysqlPodSpecUpdater{}
	checkExpectations(update.UpdateCR(updater), updater)
}

func checkExpectations(err error, updater *mysqlPodSpecUpdater) {
	t.Logs.Infof("Verifies that an update to the MySQL overrides containing a podSpec value issues a warning to " +
		"the user; also makes an illegal edit to avoid mutating the system but also generate the warning")
	if err != nil {
		t.Logs.Infof("Update error: %s", err.Error())
	}
	if updater.hasWarnings() {
		for _, warning := range updater.warnings {
			t.Logs.Infof("Warning: %v", warning)
		}
	}
	Expect(err).ToNot(BeNil())
	Expect(updater.hasWarnings()).To(BeTrue())
	Expect(updater.hasWarningText(warningSubstring))
}
