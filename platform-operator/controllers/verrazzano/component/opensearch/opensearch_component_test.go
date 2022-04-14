// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package opensearch

import (
	"context"
	"os/exec"
	"testing"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stretchr/testify/assert"
	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles"

var dnsComponents = vzapi.ComponentSpec{
	DNS: &vzapi.DNSComponent{
		External: &vzapi.External{Suffix: "blah"},
	},
}

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Elasticsearch: &vzapi.ElasticsearchComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

// genericTestRunner is used to run generic OS commands with expected results
type genericTestRunner struct {
}

// Run genericTestRunner executor
func (r genericTestRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return nil, nil, nil
}

// fakeUpgrade override the upgrade function during unit tests
func fakeUpgrade(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides helmcli.HelmOverrides) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

// TestPreUpgrade tests the Opensearch PreUpgrade call
// GIVEN an Opensearch component
//  WHEN I call PreUpgrade with defaults
//  THEN no error is returned
func TestPreUpgrade(t *testing.T) {
	// The actual pre-upgrade testing is performed by the underlying unit tests, this just adds coverage
	// for the Component interface hook
	config.TestHelmConfigDir = "../../../../helm_config"
	err := NewComponent().PreUpgrade(spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), &vzapi.Verrazzano{}, false))
	assert.NoError(t, err)
}

// TestPreInstall tests the Opensearch PreInstall call
// GIVEN an Opensearch component
//  WHEN I call PreInstall when dependencies are met
//  THEN no error is returned
func TestPreInstall(t *testing.T) {
	c := createPreInstallTestClient()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, false)
	err := NewComponent().PreInstall(ctx)
	assert.NoError(t, err)
}

// TestInstall tests the Opensearch Install call
// GIVEN an Opensearch component
//  WHEN I call Install when dependencies are met
//  THEN no error is returned
func TestInstall(t *testing.T) {
	c := createPreInstallTestClient()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
	}, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetUpgradeFunc(fakeUpgrade)
	defer helm.SetDefaultUpgradeFunc()
	helmcli.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helmcli.ChartStatusDeployed, nil
	})
	defer helmcli.SetDefaultChartStateFunction()
	err := NewComponent().Install(ctx)
	assert.NoError(t, err)
}

// TestPostInstall tests the Opensearch PostInstall call
// GIVEN an Opensearch component
//  WHEN I call PostInstall
//  THEN no error is returned
func TestPostInstall(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
	}, false)
	vzComp := NewComponent()

	// PostInstall will fail because the expected VZ ingresses are not present in cluster
	err := vzComp.PostInstall(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)

	// now get all the ingresses for VZ and add them to the fake K8S and ensure that PostInstall succeeds
	// when all the ingresses are present in the cluster
	vzIngressNames := vzComp.(opensearchComponent).GetIngressNames(ctx)
	for _, ingressName := range vzIngressNames {
		_ = c.Create(context.TODO(), &v1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName.Name, Namespace: ingressName.Namespace},
		})
	}
	for _, certName := range vzComp.(opensearchComponent).GetCertificateNames(ctx) {
		time := metav1.Now()
		_ = c.Create(context.TODO(), &certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certName.Name, Namespace: certName.Namespace},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
				},
			},
		})
	}
	err = vzComp.PostInstall(ctx)
	assert.NoError(t, err)
}

// TestPostInstallCertsNotReady tests the Opensearch PostInstall call
// GIVEN an Opensearch component
//  WHEN I call PostInstall and the certificates aren't ready
//  THEN a retryable error is returned
func TestPostInstallCertsNotReady(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: dnsComponents,
		},
	}, false)
	vzComp := NewComponent()

	// PostInstall will fail because the expected VZ ingresses are not present in cluster
	err := vzComp.PostInstall(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)

	// now get all the ingresses for VZ and add them to the fake K8S and ensure that PostInstall succeeds
	// when all the ingresses are present in the cluster
	vzIngressNames := vzComp.(opensearchComponent).GetIngressNames(ctx)
	for _, ingressName := range vzIngressNames {
		_ = c.Create(context.TODO(), &v1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName.Name, Namespace: ingressName.Namespace},
		})
	}
	for _, certName := range vzComp.(opensearchComponent).GetCertificateNames(ctx) {
		time := metav1.Now()
		_ = c.Create(context.TODO(), &certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certName.Name, Namespace: certName.Namespace},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
				},
			},
		})
	}
	err = vzComp.PostInstall(ctx)

	expectedErr := spi2.RetryableError{
		Source:    vzComp.Name(),
		Operation: "Check if certificates are ready",
	}
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestGetCertificateNames tests the Opensearch GetCertificateNames call
// GIVEN an Opensearch component
//  WHEN I call GetCertificateNames
//  THEN the correct number of certificate names are returned based on what is enabled
func TestGetCertificateNames(t *testing.T) {
	vmiEnabled := false
	vz := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{Suffix: "blah"},
				},
				Grafana:       &vzapi.GrafanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &vmiEnabled}},
				Prometheus:    &vzapi.PrometheusComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &vmiEnabled}},
				Kibana:        &vzapi.KibanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &vmiEnabled}},
				Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &vmiEnabled},
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vz, false)
	vzComp := NewComponent()

	certNames := vzComp.GetCertificateNames(ctx)
	assert.Len(t, certNames, 1, "Unexpected number of cert names")

	vmiEnabled = true
	vz.Spec.Components.Grafana.Enabled = &vmiEnabled
	vz.Spec.Components.Prometheus.Enabled = &vmiEnabled
	vz.Spec.Components.Kibana.Enabled = &vmiEnabled
	vz.Spec.Components.Elasticsearch.Enabled = &vmiEnabled

	certNames = vzComp.GetCertificateNames(ctx)
	assert.Len(t, certNames, 4, "Unexpected number of cert names")
}

// TestUpgrade tests the Opensearch Upgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN an Opensearch component upgrading from 1.1.0 to 1.2.0
//  WHEN I call Upgrade
//  THEN no error is returned
func TestUpgrade(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Version:    "v1.2.0",
			Components: dnsComponents,
		},
		Status: vzapi.VerrazzanoStatus{Version: "1.1.0"},
	}, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	helmcli.SetCmdRunner(genericTestRunner{})
	defer helmcli.SetDefaultRunner()
	helm.SetUpgradeFunc(fakeUpgrade)
	defer helm.SetDefaultUpgradeFunc()
	helmcli.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helmcli.ChartStatusDeployed, nil
	})
	defer helmcli.SetDefaultChartStateFunction()
	err := NewComponent().Upgrade(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the Opensearch PostUpgrade call; simple wrapper exercise, more detailed testing is done elsewhere
// GIVEN an Opensearch component upgrading from 1.1.0 to 1.2.0
//  WHEN I call PostUpgrade
//  THEN no error is returned
func TestPostUpgrade(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Version:    "v1.2.0",
			Components: dnsComponents,
		},
		Status: vzapi.VerrazzanoStatus{Version: "1.1.0"},
	}, false)
	err := NewComponent().PostUpgrade(ctx)
	assert.NoError(t, err)
}

func createPreInstallTestClient(extraObjs ...client.Object) client.Client {
	objs := []client.Object{}
	objs = append(objs, extraObjs...)
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(objs...).Build()
	return c
}

// TestIsEnabledNilOpensearch tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Opensearch component is enabled
//  THEN true is returned
func TestIsEnabledNilOpensearch(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Elasticsearch = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Opensearch component is nil
//  THEN true is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Opensearch component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Elasticsearch.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Opensearch component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Elasticsearch.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Opensearch component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Elasticsearch.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

func getBoolPtr(b bool) *bool {
	return &b
}

func Test_opensearchComponent_ValidateUpdate(t *testing.T) {
	disabled := false
	var pvc1Gi, _ = resource.ParseQuantity("1Gi")
	var pvc2Gi, _ = resource.ParseQuantity("2Gi")
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
						Elasticsearch: &vzapi.ElasticsearchComponent{
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
						Elasticsearch: &vzapi.ElasticsearchComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-installargs",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{{Name: "foo", Value: "bar"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "emptyDir to PVC in defaultVolumeSource",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "PVC to emptyDir in volumeSource",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "resize pvc in defaultVolumeSource",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"},
					},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc1Gi,
									},
								},
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"},
					},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc2Gi,
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "resize pvc in ESInstallArgs",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{
								{
									Name:  "nodes.data.requests.storage",
									Value: "1Gi",
								},
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{
								{
									Name:  "nodes.data.requests.storage",
									Value: "2Gi",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disable-opensearch",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &disabled},
					},
				},
			},
			wantErr: true,
		},
		{
			// Change to OS installargs allowed, persistence changes are supported
			name: "change-os-installargs",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{{Name: "foo", Value: "bar"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disable-grafana",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Grafana: &vzapi.GrafanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &disabled}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "disable-prometheus",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus: &vzapi.PrometheusComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &disabled}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "disable-osd",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Kibana: &vzapi.KibanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &disabled}},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			err := c.ValidateUpdate(tt.old, tt.new)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
