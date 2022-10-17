// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	admv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1 "k8s.io/client-go/applyconfigurations/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testBomFile = "../../verrazzano-bom.json"
const fakeCompReleaseName = "verrazzano-authproxy"

// TestUpdate tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a condition with type InstallStarted
func TestUpdate(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "TestUpdate"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(0)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", "1.3.0", namespace, name, "true")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateReconciling, vz.Status.State)
	asserts.True(*fakeCompUpdated)
	asserts.Equal(vz.Generation, vz.Status.Components[fakeCompReleaseName].LastReconciledGeneration)
	asserts.Equal(vzapi.CondInstallStarted, vz.Status.Components[fakeCompReleaseName].Conditions[0].Type)
	asserts.Equal(vzapi.CondInstallComplete, vz.Status.Components[fakeCompReleaseName].Conditions[1].Type)
	asserts.False(result.Requeue)
}

// TestNoUpdateSameGeneration tests the reconcile func with same generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the same LastReconciledGeneration as verrazzano CR
// THEN ensure a condition with type InstallStarted is not added
func TestNoUpdateSameGeneration(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "TestSameGeneration"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(0)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t, lastReconciledGeneration, reconcilingGen, lastReconciledGeneration,
		"1.3.1", "1.3.1", namespace, name, "true")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateReady, vz.Status.State)
	asserts.Nil(fakeCompUpdated)
	asserts.False(result.Requeue)
}

// TestUpdateWithUpgrade tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a condition with type UpgradeStarted
func TestUpdateWithUpgrade(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(0)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t, lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", "1.2.0", namespace, name, "true")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateUpgrading, vz.Status.State)
	asserts.Nil(fakeCompUpdated)
	asserts.True(result.Requeue)
}

// TestUpdateOnUpdate tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a condition with type InstallStarted
func TestUpdateOnUpdate(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(3)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t,
		reconcilingGen+1, reconcilingGen, lastReconciledGeneration,
		"1.3.3", "1.3.3", namespace, name, "true")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateReconciling, vz.Status.State)
	asserts.True(*fakeCompUpdated)
	asserts.Equal(vz.Generation, vz.Status.Components[fakeCompReleaseName].LastReconciledGeneration)
	asserts.Equal(vzapi.CondInstallStarted, vz.Status.Components[fakeCompReleaseName].Conditions[0].Type)
	asserts.Equal(vzapi.CondInstallComplete, vz.Status.Components[fakeCompReleaseName].Conditions[1].Type)
	asserts.False(result.Requeue)
}

// TestUpdateFalseMonitorChanges tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration but MonitorOverrides returns false
// THEN ensure a condition with type InstallStarted is not added
func TestUpdateFalseMonitorChanges(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "TestUpdate"
	lastReconciledGeneration := int64(2)
	reconcilingGen := int64(0)
	asserts, vz, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", "1.3.0", namespace, name, "false")
	defer reset()
	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateReady, vz.Status.State)
	asserts.Nil(fakeCompUpdated)
	asserts.False(result.Requeue)
}

func reset() {
	registry.ResetGetComponentsFn()
	config.SetDefaultBomFilePath("")
	helm.SetDefaultChartStatusFunction()
	config.SetDefaultBomFilePath("")
	helm.SetDefaultChartStatusFunction()
	config.TestProfilesDir = ""
}

func testUpdate(t *testing.T,
	//mocker *gomock.Controller, mock *mocks.MockClient,
	vzCrGen, reconcilingGen, lastReconciledGeneration int64,
	//mockStatus *mocks.MockStatusWriter,
	specVer, statusVer, namespace, name, monitorChanges string) (*assert.Assertions, *vzapi.Verrazzano, ctrl.Result, *bool, error) {
	asserts := assert.New(t)

	config.SetDefaultBomFilePath(testBomFile)

	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	fakeComp := fakeComponent{}
	fakeComp.ReleaseName = fakeCompReleaseName
	fakeComp.SupportsOperatorInstall = true
	fakeComp.monitorChanges = monitorChanges
	var fakeCompUpdated *bool
	fakeComp.installFunc = func(ctx spi.ComponentContext) error {
		update := true
		fakeCompUpdated = &update
		return nil
	}
	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComp,
		}
	})
	compStatusMap := makeVerrazzanoComponentStatusMap()
	for _, status := range compStatusMap {
		status.ReconcilingGeneration = reconcilingGen
		status.LastReconciledGeneration = lastReconciledGeneration
	}
	var vz *vzapi.Verrazzano
	// Expect a call to get the verrazzano resource.  Return resource with version

	c1, c2 := prepareContexts()

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			vz = verrazzano
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Generation: vzCrGen,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: specVer}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:   vzapi.VzStateReady,
				Version: statusVer,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
			}
			verrazzano.Status.Components = compStatusMap
			return nil
		})
	// The mocks are added to accommodate the expected calls to List instance when component is Ready
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ingressList *networkingv1.IngressList, options ...client.UpdateOption) error {
			ingressList.Items = []networkingv1.Ingress{}
			return nil
		}).AnyTimes()
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
			return nil
		}).AnyTimes()

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}
	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, labels)
	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)
	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	config.TestProfilesDir = "../../manifests/profiles"
	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(context.TODO(), request)
	mocker.Finish()
	return asserts, vz, result, fakeCompUpdated, err
}

func prepareContexts() (spi.ComponentContext, spi.ComponentContext) {
	// mock the k8s resources used in post install
	caSecret := createCASecret()
	rootCASecret := createRootCASecret()
	adminSecret := createAdminSecret()
	rancherPodList := createRancherPodListWithAllRunning()
	verrazzanoAdminClusterRole := createClusterRoles("verrazzano-admin")
	verrazzanoMonitorClusterRole := createClusterRoles("verrazzano-monitor")

	ingress := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   common.CattleSystem,
			Name:        constants.RancherIngress,
			Annotations: map[string]string{},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: "rancher",
				},
			},
		},
	}
	kcIngress := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "keycloak",
			Name:      "keycloak",
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: "keycloak",
				},
			},
		},
	}
	time := metav1.Now()
	cert := certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certificates[0].Name, Namespace: certificates[0].Namespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}
	serverURLSetting := createServerURLSetting()
	ociDriver := createOciDriver()
	okeDriver := createOkeDriver()
	authConfig := createKeycloakAuthConfig()
	localAuthConfig := createLocalAuthConfig()
	kcSecret := createKeycloakSecret()
	firstLoginSetting := createFirstLoginSetting()
	rancherPod := newPod("cattle-system", "rancher")
	rancherPod.Status = corev1.PodStatus{
		Phase: corev1.PodRunning,
	}

	clientWithoutIngress := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret, &rootCASecret, &adminSecret, &rancherPodList.Items[0], &serverURLSetting, &ociDriver, &okeDriver, &authConfig, &kcIngress, &kcSecret, &localAuthConfig, &firstLoginSetting, &verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole, rancherPod).Build()
	ctxWithoutIngress := spi.NewFakeContext(clientWithoutIngress, &vzDefaultCA, nil, false)

	clientWithIngress := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret, &rootCASecret, &adminSecret, &rancherPodList.Items[0], &ingress, &cert, &serverURLSetting, &ociDriver, &okeDriver, &authConfig, &kcIngress, &kcSecret, &localAuthConfig, &firstLoginSetting, &verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole, rancherPod).Build()
	ctxWithIngress := spi.NewFakeContext(clientWithIngress, &vzDefaultCA, nil, false)
	// mock the pod executor when resetting the Rancher admin password
	scheme.Scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"}, &corev1.PodExecOptions{})
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodExecResult = func(url *url.URL) (string, string, error) {
		var commands []string
		if commands = url.Query()["command"]; len(commands) == 3 {
			if strings.Contains(commands[2], "id,clientId") {
				return "[{\"id\":\"something\", \"clientId\":\"" + AuthConfigKeycloakClientIDRancher + "\"}]", "", nil
			}

			if strings.Contains(commands[2], "client-secret") {
				return "{\"type\":\"secret\",\"value\":\"abcdef\"}", "", nil
			}

			if strings.Contains(commands[2], "get users") {
				return "[{\"id\":\"something\", \"username\":\"verrazzano\"}]", "", nil
			}

			if strings.Contains(commands[2], fmt.Sprintf("cat %s", SettingUILogoDarkLogoFilePath)) {
				return "dark", "", nil
			}

			if strings.Contains(commands[2], fmt.Sprintf("cat %s", SettingUILogoLightLogoFilePath)) {
				return "light", "", nil
			}

		}
		return "", "", nil
	}
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config, k := k8sutilfake.NewClientsetConfig()
		return config, k, nil
	}
	return ctxWithoutIngress, ctxWithIngress
}

var (
	vzAcmeDev = vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "ACME_DEV",
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Certificate: vzapi.Certificate{
						Acme: vzapi.Acme{
							Provider:     "foobar",
							EmailAddress: "foo@bar.com",
							Environment:  "dev",
						},
					},
				},
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{Suffix: common.RancherName},
				},
			},
		},
	}
	vzDefaultCA = vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "DefaultCA",
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{Certificate: vzapi.Certificate{CA: vzapi.CA{
					SecretName:               defaultVerrazzanoName,
					ClusterResourceNamespace: defaultSecretNamespace,
				}}},
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{Suffix: common.RancherName},
				},
			},
		},
	}
)

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = networking.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = certv1.AddToScheme(scheme)
	_ = admv1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = v12.AddToScheme(scheme)
	return scheme
}

func getTestLogger(t *testing.T) vzlog.VerrazzanoLogger {
	return vzlog.DefaultLogger()
}

func createRootCASecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      common.RancherIngressCAName,
		},
		Data: map[string][]byte{
			common.RancherCACert: []byte("blahblah"),
		},
	}
}

func createCASecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultSecretNamespace,
			Name:      defaultVerrazzanoName,
		},
		Data: map[string][]byte{
			caCert: []byte("blahblah"),
		},
	}
}

func createRancherPodListWithAllRunning() v1.PodList {
	return v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod",
					Namespace: common.CattleSystem,
					Labels: map[string]string{
						"app": common.RancherName,
					},
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: "Ready", Status: "True"},
					},
				},
			},
		},
	}
}

func createClusterRoles(roleName string) rbacv1.ClusterRole {
	return rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: roleName}}
}

func createRancherPodListWithNoneRunning() v1.PodList {
	return v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod",
					Namespace: common.CattleSystem,
					Labels: map[string]string{
						"app": common.RancherName,
					},
				},
			},
		},
	}
}

func createRancherPodListWithLastRunning() v1.PodList {
	return v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod1",
					Namespace: common.CattleSystem,
					Labels: map[string]string{
						"app": common.RancherName,
					},
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: "Ready", Status: "False"},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod2",
					Namespace: common.CattleSystem,
					Labels: map[string]string{
						"app": common.RancherName,
					},
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: "Ready", Status: "True"},
					},
				},
			},
		},
	}
}

func createAdminSecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      common.RancherAdminSecret,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}
}

func createServerURLSetting() unstructured.Unstructured {
	serverURLSetting := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	serverURLSetting.SetGroupVersionKind(GVKSetting)
	serverURLSetting.SetName(SettingServerURL)
	return serverURLSetting
}

func createFirstLoginSetting() unstructured.Unstructured {
	firstLoginSetting := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	firstLoginSetting.SetGroupVersionKind(GVKSetting)
	firstLoginSetting.SetName(SettingFirstLogin)
	return firstLoginSetting
}

func createOciDriver() unstructured.Unstructured {
	ociDriver := unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"active": false,
			},
		},
	}
	ociDriver.SetGroupVersionKind(GVKNodeDriver)
	ociDriver.SetName(NodeDriverOCI)
	return ociDriver
}

func createOkeDriver() unstructured.Unstructured {
	okeDriver := unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"active": false,
			},
		},
	}
	okeDriver.SetGroupVersionKind(GVKKontainerDriver)
	okeDriver.SetName(KontainerDriverOKE)
	return okeDriver
}

func createKeycloakAuthConfig() unstructured.Unstructured {
	authConfig := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	authConfig.SetGroupVersionKind(common.GVKAuthConfig)
	authConfig.SetName(common.AuthConfigKeycloak)
	return authConfig
}

func createLocalAuthConfig() unstructured.Unstructured {
	authConfig := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	authConfig.SetGroupVersionKind(common.GVKAuthConfig)
	authConfig.SetName(AuthConfigLocal)
	return authConfig
}

func createKeycloakSecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "keycloak",
			Name:      "keycloak-http",
		},
		Data: map[string][]byte{
			"password": []byte("blahblah"),
		},
	}
}
