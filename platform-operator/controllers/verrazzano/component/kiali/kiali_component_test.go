// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package kiali

import (
	"context"
	"testing"

	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	certapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8s/errors"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Kiali: &vzapi.KialiComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

var testScheme = runtime.NewScheme()

const profilesRelativePath = "../../../../manifests/profiles"

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)

	_ = vzapi.AddToScheme(testScheme)
	_ = clustersv1alpha1.AddToScheme(testScheme)

	_ = istioclinet.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
	_ = certapiv1.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Kiali component is nil
//	THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilKiali tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Kiali component is nil
//	THEN true is returned
func TestIsEnabledNilKiali(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Kiali component enabled is nil
//	THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Kiali component is explicitly enabled
//	THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Kiali component is explicitly disabled
//	THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledManagedClusterProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Kiali enabled flag is nil and managed cluster profile
//	THEN false is returned
func TestIsEnabledManagedClusterProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.ManagedCluster
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledProdProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Kiali enabled flag is nil and prod profile
//	THEN false is returned
func TestIsEnabledProdProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.Prod
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledDevProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Kiali enabled flag is nil and dev profile
//	THEN false is returned
func TestIsEnabledDevProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Kiali = nil
	cr.Spec.Profile = vzapi.Dev
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestRemoveDeploymentAndService tests the removeDeploymentAndService function
// GIVEN a call to removeDeploymentAndService
//
//	WHEN the Kiali deployment and service exist with incorrect selectors
//	THEN the deployment and service are deleted
func TestRemoveDeploymentAndService(t *testing.T) {
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      kialiSystemName,
		},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": ComponentName,
					"app.kubernetes.io/name":     kialiSystemName,
				},
			},
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      kialiSystemName,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(deployment, service).Build()
	err := removeDeploymentAndService(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.Nil(t, err)
	deployment = &appv1.Deployment{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: kialiSystemName, Namespace: ComponentNamespace}, deployment)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
	service = &corev1.Service{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: kialiSystemName, Namespace: ComponentNamespace}, service)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

}

// TestRemoveDeploymentAndServiceNoMatch tests the removeDeploymentAndService function
// GIVEN a call to removeDeploymentAndService
//
//	WHEN the Kiali deployment and service exist with correct selectors
//	THEN the deployment and service are not deleted
func TestRemoveDeploymentAndServiceNoMatch(t *testing.T) {
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      kialiSystemName,
		},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance": kialiSystemName,
					"app.kubernetes.io/name":     "kiali",
				},
			},
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      kialiSystemName,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(deployment, service).Build()
	err := removeDeploymentAndService(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.Nil(t, err)
	deployment = &appv1.Deployment{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: kialiSystemName, Namespace: ComponentNamespace}, deployment)
	assert.Nil(t, err)
	service = &corev1.Service{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: kialiSystemName, Namespace: ComponentNamespace}, service)
	assert.Nil(t, err)
}

// TestKialiPostInstallUpdateResources tests the PostInstall function
// GIVEN a call to PostInstall
//
//	WHEN the Kiali resources already exist
//	THEN no error is returned
func TestKialiPostInstallUpdateResources(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.KialiIngress, Namespace: constants.VerrazzanoSystemNamespace},
	}

	time := metav1.Now()
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certificates[0].Name, Namespace: certificates[0].Namespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}
	authPol := &istioclisec.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali-authzpol"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(ingress, authPol, cert).Build()
	err := NewComponent().PostInstall(spi.NewFakeContext(fakeClient, vz, nil, false))
	assert.Nil(t, err)
}

// TestKialiPostInstallCreateResources tests the PostInstall function
// GIVEN a call to PostInstall
//
//	WHEN the Kiali ingress and authpolicies don't yet exist
//	THEN no error is returned
func TestKialiPostInstallCreateResources(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}

	time := metav1.Now()
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certificates[0].Name, Namespace: certificates[0].Namespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(cert).Build()
	err := NewComponent().PostInstall(spi.NewFakeContext(fakeClient, vz, nil, false))
	ing := v1.Ingress{}
	// Kiali ingress should be created
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: constants.KialiIngress, Namespace: constants.VerrazzanoSystemNamespace}, &ing))
	assert.Nil(t, err)
}

// TestKialiPostUpgradeUpdateResources tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//
//	WHEN the Kiali resources exist
//	THEN no error is returned
func TestKialiPostUpgradeUpdateResources(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: kialiSystemName, Namespace: constants.VerrazzanoSystemNamespace},
	}
	authPol := &istioclisec.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali-authzpol"},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(ingress, authPol).Build()
	err := NewComponent().PostUpgrade(spi.NewFakeContext(fakeClient, vz, nil, false))
	assert.Nil(t, err)
}

// TestKialiPostUninstall tests the PostUninstall function
// GIVEN a call to PostUninstall
//
//		WHEN the Kiali ingress exists
//		  THEN it is deleted
//	 WHEN no Kiali ingress is present
//	   THEN no error is returned
func TestKialiPostUninstall(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: kialiSystemName, Namespace: constants.VerrazzanoSystemNamespace},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(ingress).Build()
	err := NewComponent().PostUninstall(spi.NewFakeContext(fakeClient, vz, nil, true))
	assert.Nil(t, err)
	ingAfterPostUninstall := v1.Ingress{}
	// ingress should not exist after postuninstall
	assert.True(t,
		errors.IsNotFound(
			fakeClient.Get(
				context.TODO(),
				types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace},
				&ingAfterPostUninstall)))

	// WHEN no Kiali ingress is present
	// THEN no error is returned
	fakeClientNoIngress := fake.NewClientBuilder().WithScheme(testScheme).Build()
	err = NewComponent().PostUninstall(spi.NewFakeContext(fakeClientNoIngress, vz, nil, true))
	assert.Nil(t, err)
}

// TestPreUpgrade tests the Kiali PreUpgrade call
// GIVEN a Kiali component
//
//	WHEN I call PreUpgrade with defaults
//	THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helmcli.CreateActionConfig(true, kialiSystemName, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
			now := time.Now()
			return &release.Release{
				Name:      kialiSystemName,
				Namespace: ComponentNamespace,
				Info: &release.Info{
					FirstDeployed: now,
					LastDeployed:  now,
					Status:        releaseStatus,
					Description:   "Named Release Stub",
				},
				Version: 1,
			}
		})
	})

	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = "../../../../thirdparty"
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      kialiSystemName,
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      kialiSystemName,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(deployment, service).Build()
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fakeClient, nil, nil, false))
	assert.NoError(t, err)
}

func getBoolPtr(b bool) *bool {
	return &b
}

func Test_kialiComponent_ValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Kiali: &vzapi.KialiComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Kiali: &vzapi.KialiComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestKialiComponentValidateUpdateV1Beta1 tests the ValidateUpdateV1Beta1 function
// GIVEN an old and new v1beta1 VZ resource
// WHEN a call to ValidateUpdateV1Beta1 is made
// THEN return an error if the component is being disabled
func TestKialiComponentValidateUpdateV1Beta1(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *installv1beta1.Verrazzano
		new     *installv1beta1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &installv1beta1.Verrazzano{
				Spec: installv1beta1.VerrazzanoSpec{
					Components: installv1beta1.ComponentSpec{
						Kiali: &installv1beta1.KialiComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &installv1beta1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &installv1beta1.Verrazzano{},
			new: &installv1beta1.Verrazzano{
				Spec: installv1beta1.VerrazzanoSpec{
					Components: installv1beta1.ComponentSpec{
						Kiali: &installv1beta1.KialiComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &installv1beta1.Verrazzano{},
			new:     &installv1beta1.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdateV1Beta1(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestKialiComponentMonitorOverrides tests the MonitorOverrides function
// GIVEN an effective VZ CR
// WHEN a call to MonitorOverrides is made
// THEN return true if monitoring of install overrides is enabled, false otherwise
func TestKialiComponentMonitorOverrides(t *testing.T) {
	c := NewComponent()
	cli := fake.NewClientBuilder().WithScheme(testScheme).Build()
	tests := []struct {
		name   string
		vz     *vzapi.Verrazzano
		result bool
	}{
		{
			"Kiali Component is nil",
			&vzapi.Verrazzano{},
			false,
		},
		{
			"Kiali component initialised",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Kiali: &vzapi.KialiComponent{},
					},
				},
			},
			true,
		},
		{
			"MonitorChanges explicitly enabled in Kiali component",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Kiali: &vzapi.KialiComponent{
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: getBoolPtr(true)},
						},
					},
				},
			},
			true,
		},
		{
			"MonitorChanges explicitly disabled in Kiali component",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Kiali: &vzapi.KialiComponent{
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: getBoolPtr(false)},
						},
					},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(cli, tt.vz, nil, false)
			result := c.MonitorOverrides(ctx)
			assert.Equal(t, tt.result, result)
		})
	}
}
