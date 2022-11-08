// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	admv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynfake "k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const foo = "foo"

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
	serverURLSetting.SetGroupVersionKind(common.GVKSetting)
	serverURLSetting.SetName(SettingServerURL)
	return serverURLSetting
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

// TestUseAdditionalCAs verifies that additional CAs should be used when specified in the Verrazzano CR
// GIVEN a Verrazzano CR
//
//	WHEN useAdditionalCAs is called
//	THEN useAdditionalCAs return true or false if additional CAs are required
func TestUseAdditionalCAs(t *testing.T) {
	var tests = []struct {
		in  vzapi.Acme
		out bool
	}{
		{vzapi.Acme{Environment: "dev"}, true},
		{vzapi.Acme{Environment: "production"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.in.Environment, func(t *testing.T) {
			assert.Equal(t, tt.out, useAdditionalCAs(tt.in))
		})
	}
}

// TestGetRancherHostname verifies the Rancher hostname can be generated
// GIVEN a Verrazzano CR
//
//	WHEN getRancherHostname is called
//	THEN getRancherHostname should return the Rancher hostname
func TestGetRancherHostname(t *testing.T) {
	expected := fmt.Sprintf("%s.%s.rancher", common.RancherName, vzAcmeDev.Spec.EnvironmentName)
	actual, _ := getRancherHostname(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzAcmeDev)
	assert.Equal(t, expected, actual)
}

// TestGetRancherHostnameNotFound verifies the Rancher hostname can not be generated in the CR is invalid
// GIVEN an invalid Verrazzano CR
//
//	WHEN getRancherHostname is called
//	THEN getRancherHostname should return an error
func TestGetRancherHostnameNotFound(t *testing.T) {
	_, err := getRancherHostname(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzapi.Verrazzano{})
	assert.NotNil(t, err)
}

// TestChartsNotUpdatedWorkaround tests the chartsNotUpdatedWorkaround function
// GIVEN an existing Rancher installation
//
//	WHEN chartsNotUpdatedWorkaround is called
//	THEN the Rancher deployment has been scaled down and the ClusterRepo resources for system charts are deleted
func TestChartsNotUpdatedWorkaround(t *testing.T) {
	// the first pass will have the Rancher deployment available replicas set to 3
	client := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: common.CattleSystem,
				Name:      common.RancherName,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 3,
			},
		},
	).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)
	err := chartsNotUpdatedWorkaround(ctx)
	assert.Error(t, err)

	// create a fake dynamic client to serve the Setting and ClusterRepo resources
	fakeDynamicClient := dynfake.NewSimpleDynamicClient(getScheme(), newClusterRepoResources()...)

	// override the getDynamicClientFunc for unit testing and reset it when done
	prevGetDynamicClientFunc := getDynamicClientFunc
	getDynamicClientFunc = func() (dynamic.Interface, error) { return fakeDynamicClient, nil }
	defer func() {
		getDynamicClientFunc = prevGetDynamicClientFunc
	}()

	// the second pass now shows the available replicas to be zero
	client = fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: common.CattleSystem,
				Name:      common.RancherName,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
			},
		},
	).Build()
	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)
	err = chartsNotUpdatedWorkaround(ctx)
	assert.NoError(t, err)

	// validate that the Setting and ClusterRepo resources have been deleted
	_, err = fakeDynamicClient.Resource(cattleSettingsGVR).Get(context.TODO(), chartDefaultBranchName, metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err))

	_, err = fakeDynamicClient.Resource(cattleClusterReposGVR).Get(context.TODO(), rancherChartsClusterRepoName, metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err))
	_, err = fakeDynamicClient.Resource(cattleClusterReposGVR).Get(context.TODO(), rancherPartnerChartsClusterRepoName, metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err))
	_, err = fakeDynamicClient.Resource(cattleClusterReposGVR).Get(context.TODO(), rancherRke2ChartsClusterRepoName, metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err))

	// this ClusterRepo should not have been deleted
	_, err = fakeDynamicClient.Resource(cattleClusterReposGVR).Get(context.TODO(), "app-charts", metav1.GetOptions{})
	assert.NoError(t, err)
}

// newClusterRepoResources creates resources that will be loaded into the dynamic k8s client
func newClusterRepoResources() []runtime.Object {
	cattleSettings := &unstructured.Unstructured{}
	cattleSettings.SetGroupVersionKind(common.GVKSetting)
	cattleSettings.SetName(chartDefaultBranchName)

	gvk := schema.GroupVersionKind{Group: "catalog.cattle.io", Version: "v1", Kind: "ClusterRepo"}
	rancherClusterRepo := &unstructured.Unstructured{}
	rancherClusterRepo.SetGroupVersionKind(gvk)
	rancherClusterRepo.SetName(rancherChartsClusterRepoName)

	rancherPartnerClusterRepo := &unstructured.Unstructured{}
	rancherPartnerClusterRepo.SetGroupVersionKind(gvk)
	rancherPartnerClusterRepo.SetName(rancherPartnerChartsClusterRepoName)

	rancherRke2ClusterRepo := &unstructured.Unstructured{}
	rancherRke2ClusterRepo.SetGroupVersionKind(gvk)
	rancherRke2ClusterRepo.SetName(rancherRke2ChartsClusterRepoName)

	appClusterRepo := &unstructured.Unstructured{}
	appClusterRepo.SetGroupVersionKind(gvk)
	appClusterRepo.SetName("app-charts")

	return []runtime.Object{cattleSettings, rancherClusterRepo, rancherPartnerClusterRepo, rancherRke2ClusterRepo, appClusterRepo}
}

// TestGetUserNameForPrincipal tests getUserNameForPrincipal to get the correct base32 encode
// hash string for the given principalID string
func TestGetUserNameForPrincipal(t *testing.T) {
	tests := []struct {
		name      string
		principal string
		want      string
	}{
		{
			"test getUserNameForPrincipal",
			"keycloakoidc_user://37f24158-e692-45b8-a789-61e0cb6e94f2",
			"u-mikd3gyuml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getUserNameForPrincipal(tt.principal); got != tt.want {
				t.Errorf("getUserNameForPrincipal() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetRancherUsername tests getRancherUsername to check
// WHEN Keycloak user is provided
// THEN name of the Rancher user that is mapped to the given Keycloak user is returned
func TestGetRancherUsername(t *testing.T) {

	keycloakUser := &keycloak.KeycloakUser{
		ID: "53g24158-e692-45b8-a789-61e0cb6e94f3",
	}
	principalLabel := UserPrincipalKeycloakPrefix + keycloakUser.ID
	rancherUsername := getUserNameForPrincipal(principalLabel)
	rancherUser := getFakeRancherUser(UserVerrazzano, principalLabel)
	rancherUser2 := getFakeRancherUser(rancherUsername, principalLabel)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	// Expect a call to get the Rancher user
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: UserVerrazzano}, gomock.Not(gomock.Nil())).
		Return(fmt.Errorf("internal server error"))
	tests := []struct {
		name    string
		ctx     spi.ComponentContext
		vzUser  *keycloak.KeycloakUser
		want    string
		wantErr bool
	}{
		{
			"TestGetRancherUsername with existing Rancher user u-verrazzano",
			spi.NewFakeContext(fake.NewClientBuilder().WithObjects(rancherUser).Build(), &vzapi.Verrazzano{}, nil, false),
			keycloakUser,
			UserVerrazzano,
			false,
		},
		{
			"TestGetRancherUsername with existing Rancher user u-<hash>",
			spi.NewFakeContext(fake.NewClientBuilder().WithObjects(rancherUser2).Build(), &vzapi.Verrazzano{}, nil, false),
			keycloakUser,
			rancherUsername,
			false,
		},
		{
			"TestGetRancherUsername with no existing Rancher user",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false),
			keycloakUser,
			rancherUsername,
			false,
		},
		{
			"TestGetRancherUsername when get Rancher user API gets failed",
			spi.NewFakeContext(mock, &vzapi.Verrazzano{}, nil, false),
			keycloakUser,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRancherUsername(tt.ctx, tt.vzUser)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRancherUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getRancherUsername() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDeleteLocalCluster tests the DeleteLocalCluster function call for various scenarios
func TestDeleteLocalCluster(t *testing.T) {
	asserts := assert.New(t)
	localCluster0 := unstructured.Unstructured{}
	localCluster0.SetGroupVersionKind(GVKCluster)
	localCluster0.SetName(ClusterLocal)

	localCluster1 := unstructured.Unstructured{}
	localCluster1.SetGroupVersionKind(GVKCluster)
	localCluster1.SetName(foo)

	testLogger := vzlog.DefaultLogger()
	tests := []struct {
		lCluster unstructured.Unstructured
		name     string
		found    bool
	}{
		// GIVEN an environment with rancher cluster
		// WHEN a call to DeleteLocalCluster is made
		// THEN rancher cluster with the name local is deleted
		{
			lCluster: localCluster0,
			name:     ClusterLocal,
			found:    false,
		},
		// GIVEN an environment with rancher cluster
		// WHEN a call to DeleteLocalCluster is made
		// THEN rancher cluster with name other than local is not deleted
		{
			lCluster: localCluster1,
			name:     foo,
			found:    true,
		},
	}
	for _, tt := range tests {
		cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&tt.lCluster).Build()
		DeleteLocalCluster(testLogger, cli)
		lc := tt.lCluster
		lcName := types.NamespacedName{Name: tt.name}
		if tt.found {
			asserts.NoError(cli.Get(context.Background(), lcName, &lc))
		} else {
			asserts.True(errors.IsNotFound(cli.Get(context.Background(), lcName, &lc)))
		}
	}
}

// TestDisableFirstLogin tests the disableFirstLogin func call
func TestDisableFirstLogin(t *testing.T) {
	asserts := assert.New(t)
	fl := unstructured.Unstructured{}
	fl.SetGroupVersionKind(common.GVKSetting)
	fl.SetName(common.SettingFirstLogin)

	fooLogin := unstructured.Unstructured{}
	fooLogin.SetGroupVersionKind(common.GVKSetting)
	fooLogin.SetName(foo)

	tests := []struct {
		firstLogin  unstructured.Unstructured
		name        string
		wantErr     bool
		errContains string
	}{
		{
			// GIVEN an environment with rancher first login setting
			// WHEN a call to disableFirstLogin func is made
			// THEN the first-login value is set to false
			firstLogin:  fl,
			name:        common.SettingFirstLogin,
			wantErr:     false,
			errContains: "",
		},
		// GIVEN an environment without rancher first login setting
		// WHEN a call to disableFirstLogin func is made
		// THEN an error is returned complaining about the first login setting
		{
			firstLogin:  fooLogin,
			name:        foo,
			wantErr:     true,
			errContains: "Failed getting first-login setting",
		},
	}
	for _, tt := range tests {
		cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&tt.firstLogin).Build()
		fakeCtx := spi.NewFakeContext(cli, &vzapi.Verrazzano{}, nil, false)
		err := disableFirstLogin(fakeCtx)
		if tt.wantErr {
			asserts.ErrorContains(err, tt.errContains)
		} else {
			asserts.NoError(cli.Get(context.Background(), types.NamespacedName{Name: tt.name}, &fl))
			asserts.Equal(fl.UnstructuredContent()["value"], "false")
		}
	}
}

// TestCreateOrUpdateRancherVerrazzanoUserGlobalRoleBinding tests the following scenario
// GIVEN a call to create or update Verrazzano user admin
// WHEN the func call executes successfully
// THEN the Rancher Verrazzano user admin is created successfully with correct data
func TestCreateOrUpdateRancherVerrazzanoUserGlobalRoleBinding(t *testing.T) {
	asserts := assert.New(t)
	cli := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	fakeCtx := spi.NewFakeContext(cli, &vzapi.Verrazzano{}, nil, false)

	asserts.NoError(createOrUpdateRancherVerrazzanoUserGlobalRoleBinding(fakeCtx, foo))

	obj := unstructured.Unstructured{}
	obj.SetGroupVersionKind(GVKGlobalRoleBinding)
	nsn := types.NamespacedName{Name: GlobalRoleBindingVerrazzanoPrefix + foo}
	asserts.NoError(cli.Get(context.Background(), nsn, &obj))

	data := obj.UnstructuredContent()
	asserts.Equal(AdminRoleName, data[GlobalRoleBindingAttributeRoleName])
	asserts.Equal(foo, data[GlobalRoleBindingAttributeUserName])
}

// TestCreateOrUpdateRancherVerrazzanoUser tests the following scenario
// GIVEN a call to create or update Verrazzano user
// WHEN the func call executes successfully
// THEN the Rancher Verrazzano user is created successfully with correct data
func TestCreateOrUpdateRancherVerrazzanoUser(t *testing.T) {
	asserts := assert.New(t)

	fakeVzUser := &keycloak.KeycloakUser{
		ID:       foo,
		Username: foo,
	}
	cli := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	fakeCtx := spi.NewFakeContext(cli, &vzapi.Verrazzano{}, nil, false)

	asserts.NoError(createOrUpdateRancherVerrazzanoUser(fakeCtx, fakeVzUser, foo))
	checkVzUser := unstructured.Unstructured{}
	checkVzUser.SetGroupVersionKind(GVKUser)
	asserts.NoError(cli.Get(context.Background(), types.NamespacedName{Name: foo}, &checkVzUser))

	data := checkVzUser.UnstructuredContent()
	caser := cases.Title(language.English)

	asserts.Equal(fakeVzUser.Username, data[UserAttributeUserName])
	asserts.Equal(caser.String(fakeVzUser.Username), data[UserAttributeDisplayName])
	asserts.Equal(caser.String(UserVerrazzanoDescription), data[UserAttributeDescription])
	asserts.Equal([]interface{}{UserPrincipalKeycloakPrefix + fakeVzUser.ID, UserPrincipalLocalPrefix + foo}, data[UserAttributePrincipalIDs])
}

// TestCreateOrUpdateClusterRoleTemplateBinding tests the following scenario
// GIVEN func call to create or update ClusterRoleTemplateBinding to add Keycloak groups to the Rancher cluster
// WHEN the call is successful
// THEN the resource is created or updated with correct data
func TestCreateOrUpdateClusterRoleTemplateBinding(t *testing.T) {
	asserts := assert.New(t)
	cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ClusterLocal,
			},
		},
	).Build()
	fakeCtx := spi.NewFakeContext(cli, &vzapi.Verrazzano{}, nil, false)

	asserts.NoError(createOrUpdateClusterRoleTemplateBinding(fakeCtx, foo, foo))
	obj := unstructured.Unstructured{}
	obj.SetGroupVersionKind(GVKClusterRoleTemplateBinding)
	nsn := types.NamespacedName{Name: "crtb-foo-foo", Namespace: ClusterLocal}
	asserts.NoError(cli.Get(context.Background(), nsn, &obj))

	data := obj.UnstructuredContent()
	asserts.Equal(ClusterLocal, data[ClusterRoleTemplateBindingAttributeClusterName])
	asserts.Equal(GroupPrincipalKeycloakPrefix+foo, data[ClusterRoleTemplateBindingAttributeGroupPrincipalName])
	asserts.Equal(foo, data[ClusterRoleTemplateBindingAttributeRoleTemplateName])
}

// TestCreateOrUpdateRoleTemplate tests the following scenario
// GIVEN func call to create or update ClusterRoleTemplateBinding to add Keycloak groups to the Rancher cluster
// WHEN the call is successful
// THEN the resource is created or updated with correct data
func TestCreateOrUpdateRoleTemplate(t *testing.T) {
	asserts := assert.New(t)
	clr := rbacv1.ClusterRole{}
	clrName := foo + "-" + foo
	clr.SetName(clrName)
	tests := []struct {
		skipClusterRole bool
		wantErr         bool
		errContains     string
	}{
		{
			skipClusterRole: true,
			wantErr:         true,
			errContains:     "failed creating RoleTemplate, unable to fetch ClusterRole",
		},
		{
			skipClusterRole: false,
			wantErr:         false,
			errContains:     "",
		},
	}

	for _, tt := range tests {
		cliBuilder := fake.NewClientBuilder().WithScheme(getScheme()).WithScheme(getScheme())
		var cli client.WithWatch
		if tt.skipClusterRole {
			cli = cliBuilder.Build()
		} else {
			cli = cliBuilder.WithObjects(&clr).Build()
		}
		fakeCtx := spi.NewFakeContext(cli, &vzapi.Verrazzano{}, nil, false)
		err := createOrUpdateRoleTemplate(fakeCtx, clrName)
		if tt.wantErr {
			asserts.ErrorContains(err, tt.errContains)
		} else {
			obj := unstructured.Unstructured{}
			obj.SetGroupVersionKind(GVKRoleTemplate)
			asserts.NoError(cli.Get(context.Background(), types.NamespacedName{Name: clrName}, &obj))
			data := obj.UnstructuredContent()
			asserts.False(data[RoleTemplateAttributeBuiltin].(bool))
			asserts.Equal("cluster", data[RoleTemplateAttributeContext])
			caser := cases.Title(language.English)
			asserts.Equal(caser.String(strings.Replace(clrName, "-", " ", 1)), data[RoleTemplateAttributeDisplayName])
			asserts.True(data[RoleTemplateAttributeExternal].(bool))
			asserts.True(data[RoleTemplateAttributeHidden].(bool))
		}
	}
}

// getFakeRancherUser constructs a fake unstructured Rancher user object
func getFakeRancherUser(userName string, principal string) *unstructured.Unstructured {
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(GVKUser)
	resource.SetName(userName)
	resource.SetNamespace("")
	data := resource.UnstructuredContent()
	data[UserAttributeUserName] = userName
	caser := cases.Title(language.English)
	data[UserAttributeDisplayName] = caser.String(userName)
	data[UserAttributeDescription] = caser.String(UserVerrazzanoDescription)
	data[UserAttributePrincipalIDs] = []interface{}{principal, UserPrincipalLocalPrefix + userName}
	return resource
}
