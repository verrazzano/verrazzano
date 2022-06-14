// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestUpgradeCmdDefaultNoWait
// GIVEN a CLI upgrade command with all defaults and --wait==false
//  WHEN I call cmd.Execute for upgrade
//  THEN the CLI upgrade command is successful
func TestUpgradeCmdDefaultNoWait(t *testing.T) {
	vpo := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app": constants.VerrazzanoPlatformOperator,
			},
		},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "webhook-init",
					Ready: true,
				},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "verrazzano-platform-operator",
					Ready: true,
				},
			},
		},
	}
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vpo, vz).Build()

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	rc.SetClient(c)
	cmd := NewCmdUpgrade(rc)
	assert.NotNil(t, cmd)
	cmd.PersistentFlags().Set(constants.WaitFlag, "false")

	// Run upgrade command
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", errBuf.String())
}
