// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package configmap

import (
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

const (
	testWorkload1 = "testWorkload1"
	testWorkload2 = "testWorkload2"
	testCMName    = "test-CM-Name"
)

var testMWC = admissionv1.MutatingWebhookConfiguration{
	ObjectMeta: metav1.ObjectMeta{
		Name:      mutatingWebhookConfigName,
		Namespace: constants.VerrazzanoSystemNamespace,
	},
	Webhooks: []admissionv1.MutatingWebhook{
		{
			Name: WebhookName,
			Rules: []admissionv1.RuleWithOperations{
				{
					Operations: nil,
					Rule: admissionv1.Rule{
						Resources: defaultResourceList,
						Scope:     nil,
					},
				},
			},
		},
	},
}

var testConfigMap = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      testCMName,
		Namespace: constants.VerrazzanoSystemNamespace,
	},
	Data: map[string]string{
		resourceIdentifier: testWorkload1 + "\n" + testWorkload2,
	},
}

// TestGetWorkloadWebhook tests the creation process of the metrics template
// GIVEN a MutatingWebhookConfiguration
// WHEN the function is called
// THEN return the webhook with the correct name
func TestGetWorkloadWebhook(t *testing.T) {
	assert := asserts.New(t)

	webhook := getWorkloadWebhook(&testMWC)
	assert.Equal(testMWC.Webhooks[0], *webhook)
}

// TestFormatWorkloadResources tests the formatting of the workload list
// GIVEN a ConfigMap data string and a resource list
// WHEN the function is called
// THEN return a list of new workload resources
func TestFormatWorkloadResources(t *testing.T) {
	assert := asserts.New(t)
	localDefaultList := make([]string, len(defaultResourceList))
	copy(localDefaultList, defaultResourceList)

	localDefaultList = formatWorkloadResources("\n  "+testWorkload1+"  \n   "+testWorkload2+"   ", localDefaultList)
	assert.Equal(append(defaultResourceList, strings.ToLower(testWorkload1), strings.ToLower(testWorkload2)), localDefaultList)
}

// TestGenerateUniqueMap tests the generation of unique values from the resource list
// GIVEN a list of strings
// WHEN the function is called
// THEN return a map of all unique values mapping to true
func TestGenerateUniqueMap(t *testing.T) {
	assert := asserts.New(t)
	testMap := map[string]bool{}
	for _, resource := range defaultResourceList {
		testMap[resource] = true
	}

	mutantList := append(defaultResourceList, defaultResourceList...)

	uniqueMap := generateUniqueMap(mutantList)
	assert.Equal(testMap, uniqueMap)
}
