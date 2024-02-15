// Copyright (c) 2023, 2024 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"context"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"os"
	"testing"

	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	adminv1 "k8s.io/api/admissionregistration/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmdHelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestInstallProgressFlag
// GIVEN a CLI install command with progress option enabled
//
//	WHEN I call cmd.Execute for install
//	THEN the CLI install command is successful
func TestInstallProgressFlag(t *testing.T) {
	vz1 := v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "admin",
		},
		Spec: v1beta1.VerrazzanoSpec{
			Profile: v1beta1.Dev,
		},
		Status: v1beta1.VerrazzanoStatus{
			Version:    "v1.6.2",
			Conditions: nil,
			State:      v1beta1.VzStateReconciling,
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz1).Build()
	cmd, rc := createNewTestCommandAndContext(t, c)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd.PersistentFlags().Set(constants.ProgressFlag, "true")
	cmd.PersistentFlags().Set(constants.TimeoutFlag, "3m")
	tempKubeConfigPath, _ := os.CreateTemp(os.TempDir(), testKubeConfig)
	cmd.Flags().String(constants.GlobalFlagKubeConfig, tempKubeConfigPath.Name(), "")
	cmd.Flags().String(constants.GlobalFlagContext, testK8sContext, "")
	cmd.PersistentFlags().Set(constants.VersionFlag, "v1.6.2")
	c.Create(context.TODO(), &vz1)
	cmdHelpers.SetDeleteFunc(cmdHelpers.FakeDeleteFunc)
	defer cmdHelpers.SetDefaultDeleteFunc()
	cmdHelpers.SetVPOIsReadyFunc(func(_ client.Client) (bool, error) { return true, nil })
	defer cmdHelpers.SetDefaultVPOIsReadyFunc()

	SetValidateCRFunc(FakeValidateCRFunc)
	defer SetDefaultValidateCRFunc()

	// Run install command
	err := cmd.Execute()
	assert.NoError(t, err)
	errBytes, err := os.ReadFile(rc.ErrOut.Name())
	assert.NoError(t, err)
	assert.Equal(t, "", string(errBytes))
	vz := v1beta1.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "admin"}, &vz)
	assert.NoError(t, err)
	outBuf, err := os.ReadFile(rc.Out.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(outBuf), "Installing Verrazzano version")
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
func ensureResourcesDeleted(t *testing.T, client client.Client) {
	// Expect the Verrazzano resource to be deleted
	v := vzapi.Verrazzano{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &v)
	assert.True(t, errors.IsNotFound(err))

	// Expect the install namespace to be deleted
	ns := corev1.Namespace{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: vzconstants.VerrazzanoInstallNamespace}, &ns)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Validating Webhook Configuration to be deleted
	vwc := adminv1.ValidatingWebhookConfiguration{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperatorWebhook}, &vwc)
	assert.True(t, errors.IsNotFound(err))

	// Expect the Cluster Role Binding to be deleted
	crb := rbacv1.ClusterRoleBinding{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoPlatformOperator}, &crb)
	assert.True(t, errors.IsNotFound(err))

	// Expect the managed cluster Cluster Role to be deleted
	cr := rbacv1.ClusterRole{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoManagedCluster}, &cr)
	assert.True(t, errors.IsNotFound(err))

	// Expect the cluster Registrar Cluster Role to be deleted
	cr = rbacv1.ClusterRole{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: vzconstants.VerrazzanoClusterRancherName}, &cr)
	assert.True(t, errors.IsNotFound(err))
}
