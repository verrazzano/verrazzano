// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"bytes"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/templates"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	name          = "verrazzano"
	namespace     = "test"
	version       = "1.2.3"
	consoleURL    = "https://verrazzano.default.10.107.141.8.nip.io"
	keycloakURL   = "https://keycloak.default.10.107.141.8.nip.io"
	rancherURL    = "https://rancher.default.10.107.141.8.nip.io"
	osURL         = "https://elasticsearch.vmi.system.10.107.141.8.nip.io"
	osdURL        = "https://kibana.vmi.system.10.107.141.8.nip.io"
	grafanaURL    = "https://grafana.vmi.system.10.107.141.8.nip.io"
	prometheusURL = "https://prometheus.vmi.system.10.107.141.8.nip.io"
	kialiURL      = "https://kiali.vmi.system.10.107.141.8.nip.io"
	jaegerURL     = "https://jaeger.default.10.107.141.8.nip.io"
)

// TestStatusCmd tests the status command
// GIVEN an environment with a single VZ resource
//
//	WHEN I run the command vz status
//	THEN expect a successful status report
func TestStatusCmd(t *testing.T) {
	vz := v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1beta1.VerrazzanoSpec{
			Profile: v1beta1.Dev,
		},
		Status: v1beta1.VerrazzanoStatus{
			Version: version,
			VerrazzanoInstance: &v1beta1.InstanceInfo{
				ConsoleURL:              &consoleURL,
				KeyCloakURL:             &keycloakURL,
				RancherURL:              &rancherURL,
				OpenSearchURL:           &osURL,
				OpenSearchDashboardsURL: &osdURL,
				GrafanaURL:              &grafanaURL,
				PrometheusURL:           &prometheusURL,
				KialiURL:                &kialiURL,
				JaegerURL:               &jaegerURL,
			},
			Conditions: nil,
			State:      v1beta1.VzStateReconciling,
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz).Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()

	templateInput := TemplateInput{
		Endpoints:           getEndpoints(vz.Status.VerrazzanoInstance),
		Components:          getComponents(vz.Status.Components),
		ComponentsEnabled:   false,
		Name:                name,
		Namespace:           namespace,
		Version:             version,
		State:               string(vzapi.VzStateReconciling),
		Profile:             string(vzapi.Dev),
		AvailableComponents: getAvailableComponents(vz.Status.Available),
	}
	expectedResult, err := templates.ApplyTemplate(statusOutputTemplate, templateInput)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

// TestStatusCmdDefaultProfile tests the status command
// GIVEN an environment with a single VZ resource and the default prod profile
//
//	WHEN I run the command vz status
//	THEN expect a successful status report
func TestStatusCmdDefaultProfile(t *testing.T) {
	vz := v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: v1beta1.VerrazzanoStatus{
			Version: version,
			VerrazzanoInstance: &v1beta1.InstanceInfo{
				ConsoleURL:              &consoleURL,
				KeyCloakURL:             &keycloakURL,
				RancherURL:              &rancherURL,
				OpenSearchURL:           &osURL,
				OpenSearchDashboardsURL: &osdURL,
				GrafanaURL:              &grafanaURL,
				PrometheusURL:           &prometheusURL,
				KialiURL:                &kialiURL,
				JaegerURL:               &jaegerURL,
			},
			Conditions: nil,
			State:      v1beta1.VzStateReconciling,
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz).Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	templateInput := TemplateInput{
		Endpoints:           getEndpoints(vz.Status.VerrazzanoInstance),
		Components:          getComponents(vz.Status.Components),
		ComponentsEnabled:   false,
		Name:                name,
		Namespace:           namespace,
		Version:             version,
		State:               string(vzapi.VzStateReconciling),
		Profile:             string(vzapi.Prod),
		AvailableComponents: getAvailableComponents(vz.Status.Available),
	}
	expectedResult, err := templates.ApplyTemplate(statusOutputTemplate, templateInput)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

// TestVZNotFound tests the status command
// GIVEN an environment with a no VZ resources exist
//
//	WHEN I run the command vz status
//	THEN expect an error of no VZ resources found
func TestVZNotFound(t *testing.T) {

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects().Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Failed to find any Verrazzano resources", err.Error())
}

// TestStatusMultipleVZ tests the status command
// GIVEN an environment with a two VZ resources
//
//	WHEN I run the command vz status
//	THEN expect an error of only expecting one VZ
func TestStatusMultipleVZ(t *testing.T) {
	name := "verrazzano"
	namespace1 := "test1"
	namespace2 := "test2"

	vz1 := v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace1,
			Name:      name,
		},
	}
	vz2 := v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace2,
			Name:      name,
		},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz1, &vz2).Build()

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)

	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.Error(t, err)
	assert.Equal(t, "Expected to only find one Verrazzano resource, but found 2", err.Error())
}

func TestNilInstance(t *testing.T) {
	vz := v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Status: v1beta1.VerrazzanoStatus{
			Version:    version,
			Conditions: nil,
			State:      v1beta1.VzStateReconciling,
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz).Build()
	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	statusCmd := NewCmdStatus(rc)
	assert.NotNil(t, statusCmd)
	// Run the status command, check for the expected status results to be displayed
	err := statusCmd.Execute()
	assert.NoError(t, err)
}

func makeVerrazzanoComponentStatusMap() v1beta1.ComponentStatusMap {
	statusMap := make(v1beta1.ComponentStatusMap)
	for _, comp := range registry.GetComponents() {
		if comp.IsOperatorInstallSupported() {
			statusMap[comp.Name()] = &v1beta1.ComponentStatusDetails{
				Name: comp.Name(),
				Conditions: []v1beta1.Condition{
					{
						Type:   v1beta1.CondInstallComplete,
						Status: corev1.ConditionTrue,
					},
				},
				State: v1beta1.CompStateReady,
			}
		}
	}
	return statusMap
}
