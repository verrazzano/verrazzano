// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	networkv1 "k8s.io/api/networking/v1"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath         = "../../testdata/test_bom.json"
	testKeycloakIngressHost = "keycloak.test-env.192.132.111.122.nip.io"
	profilesRelativePath    = "../../../../manifests/profiles"
)

var keycloakClientIds = "[ {\n  \"id\" : \"a732-249893586af2\",\n  \"clientId\" : \"account\"\n}, {\n  \"id\" : \"4256-a350-e46eb48e8606\",\n  \"clientId\" : \"account-console\"\n}, {\n  \"id\" : \"4c1d-8d1b-68635e005567\",\n  \"clientId\" : \"admin-cli\"\n}, {\n  \"id\" : \"4350-ab70-17c37dd995b9\",\n  \"clientId\" : \"broker\"\n}, {\n  \"id\" : \"4f6d-a495-0e9e3849608e\",\n  \"clientId\" : \"realm-management\"\n}, {\n  \"id\" : \"4d92-9d64-f201698d2b79\",\n  \"clientId\" : \"security-admin-console\"\n}, {\n  \"id\" : \"4160-8593-32697ebf2c11\",\n  \"clientId\" : \"verrazzano-oauth-client\"\n}, {\n  \"id\" : \"bde9-9374bd6a38fd\",\n  \"clientId\" : \"verrazzano-pg\"\n}, {\n  \"id\" : \"8327-13cdbfe3b000\",\n  \"clientId\" : \"verrazzano-pkce\"\n\n}, {\n  \"id\" : \"494a-b7ec-b05681cafc73\",\n  \"clientId\" : \"webui\"\n} ]"
var keycloakErrorClientIds = "[ {\n  \"id\" : \"a732-249893586af2\",\n  \"clientId\" : \"account\"\n}, {\n  \"id\" : \"4256-a350-e46eb48e8606\",\n  \"clientId\" : \"account-console\"\n}, {\n  \"id\" : \"4c1d-8d1b-68635e005567\",\n  \"clientId\" : \"admin-cli\"\n}, {\n  \"id\" : \"4f6d-a495-0e9e3849608e\",\n  \"clientId\" : \"realm-management\"\n}, {\n  \"id\" : \"4d92-9d64-f201698d2b79\",\n  \"clientId\" : \"security-admin-console\"\n}, {\n  \"id\" : \"4160-8593-32697ebf2c11\",\n  \"clientId\" : \"verrazzano-oauth-client\"\n}, {\n  \"id\" : \"bde9-9374bd6a38fd\",\n  \"clientId\" : \"verrazzano-pg\"\n}, {\n  \"id\" : \"494a-b7ec-b05681cafc73\",\n  \"clientId\" : \"webui\"\n} ]"
var testVZ = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
	},
}
var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Keycloak: &vzapi.KeycloakComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

// fakeBash mocks a successful script run
func fakeBash(_ ...string) (string, string, error) {
	return "success", "", nil
}

// fakeBashFail mocks a failed script run
func fakeBashFail(_ ...string) (string, string, error) {
	return "fail", "Script Failed", errors.New("script Failed")
}

func fakeRESTConfig() (*rest.Config, kubernetes.Interface, error) {
	cfg, cli := k8sutilfake.NewClientsetConfig()
	return cfg, cli, nil
}

func createTestLoginSecret() *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-http",
			Namespace: "keycloak",
		},
		Data: map[string][]byte{"password": []byte("password")},
	}
}

func createTestNginxService() *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingress-controller-ingress-nginx-controller",
			Namespace: "ingress-nginx",
		},
		Spec: v1.ServiceSpec{},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{IP: "192.132.111.122",
						Hostname: ""},
				},
			},
		},
	}
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stdout, keycloakClientIds)
	os.Exit(0)
}

func fakeFailExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFailHelperProcess", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{"GO_WANT_FAIL_HELPER_PROCESS=1"}
	return cmd
}

func TestFailHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_FAIL_HELPER_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stdout, keycloakClientIds)
	os.Exit(1)
}

func fakeFailExecCommandNoClients(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFailNoClientsHelperProcess", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{"GO_WANT_FAIL_NO_CLIENTS_HELPER_PROCESS=1"}
	return cmd
}

func TestFailNoClientsHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_FAIL_NO_CLIENTS_HELPER_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stdout, "")
	os.Exit(0)
}

func fakeFailExecCommandNoUser(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFailNoUserHelperProcess", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{"GO_WANT_FAIL_NO_USER_HELPER_PROCESS=1"}
	return cmd
}

func TestFailNoUserHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_FAIL_NO_USER_HELPER_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stdout, keycloakErrorClientIds)
	os.Exit(0)
}

func fakeCreateUserGroupCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFakeCreateUserGroup", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{"GO_WANT_TEST_CREATE_USER_GROUP=1"}
	return cmd
}

func TestFakeCreateUserGroup(t *testing.T) {
	if os.Getenv("GO_WANT_TEST_CREATE_USER_GROUP") != "1" {
		return
	}

	fmt.Fprintf(os.Stdout, "Created new group with id '6653a73b-f292-4dfe-91cb-956ead33ea67'")
	os.Exit(0)
}

func fakeCreateUserGroupCommandFail(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFakeCreateUserGroupFail", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{"GO_WANT_TEST_CREATE_USER_GROUP_FAIL=1"}
	return cmd
}

func TestFakeCreateUserGroupFail(t *testing.T) {
	if os.Getenv("GO_WANT_TEST_CREATE_USER_GROUP_FAIL") != "1" {
		return
	}

	fmt.Fprintf(os.Stdout, "")
	os.Exit(0)
}

func fakeCreateUserGroupParseCommandFail(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestFakeCreateUserGroupParseFail", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	cmd.Env = []string{"GO_WANT_TEST_CREATE_USER_GROUP_PARSE_FAIL=1"}
	return cmd
}

func TestFakeCreateUserParseGroupFail(t *testing.T) {
	if os.Getenv("GO_WANT_TEST_CREATE_USER_GROUP_PARSE_FAIL") != "1" {
		return
	}

	fmt.Fprintf(os.Stdout, "Created new group with id 6653a73b-f292-4dfe-91cb-956ead33ea67")
	os.Exit(0)
}

func TestUpdateKeycloakURIs(t *testing.T) {
	k8sutil.ClientConfig = fakeRESTConfig
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	tests := []struct {
		name    string
		args    spi.ComponentContext
		wantErr bool
	}{
		{
			name: "testUpdateKeycloakURIs",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestLoginSecret(), createTestNginxService()),
				testVZ,
				false),
			wantErr: false,
		},
		{
			name: "testFailForNoKeycloakSecret",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestNginxService()),
				testVZ,
				false),
			wantErr: true,
		},
		{
			name: "testFailForKeycloakSecretPasswordEmpty",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestNginxService(), &v1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-http",
						Namespace: "keycloak",
					},
					Data: map[string][]byte{"password": []byte("")},
				}),
				testVZ,
				false),
			wantErr: true,
		},
		{
			name: "testFailForAuthenticationToKeycloak",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestLoginSecret(), createTestNginxService()),
				testVZ,
				false),
			wantErr: true,
		},
		{
			name: "testFailForNoKeycloakClientsReturned",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestLoginSecret(), createTestNginxService()),
				testVZ,
				false),
			wantErr: true,
		},
		{
			name: "testFailForKeycloakUserNotFound",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestLoginSecret(), createTestNginxService()),
				testVZ,
				false),
			wantErr: true,
		},
		{
			name: "testFailForNoIngress",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestLoginSecret()),
				testVZ,
				false),
			wantErr: true,
		},
		{
			name: "testScriptFailure",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestLoginSecret(), createTestNginxService()),
				testVZ,
				false),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCommand = fakeExecCommand
			if tt.wantErr && tt.name == "testFailForAuthenticationToKeycloak" {
				execCommand = fakeFailExecCommand
			}
			if tt.wantErr && tt.name == "testFailForNoKeycloakClientsReturned" {
				execCommand = fakeFailExecCommandNoClients
			}
			if tt.wantErr && tt.name == "testFailForKeycloakUserNotFound" {
				execCommand = fakeFailExecCommandNoUser
			}
			setBashFunc(fakeBash)
			if tt.wantErr && tt.name == "testScriptFailure" {
				setBashFunc(fakeBashFail)
			}
			defer func() { execCommand = exec.Command }()
			if err := updateKeycloakUris(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("updateKeycloakUris() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfigureKeycloakRealms tests configuration of the Keycloak realms
// GIVEN a client, and a k8s environment
// WHEN I call configureKeycloakRealms
// THEN configure the Keycloak realms, otherwise returning an error if the environment is invalid
func TestConfigureKeycloakRealms(t *testing.T) {
	loginSecret := createTestLoginSecret()
	nginxService := createTestNginxService()
	k8sutil.ClientConfig = fakeRESTConfig
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor

	var tests = []struct {
		name        string
		c           client.Client
		stdout      string
		isErr       bool
		errContains string
	}{
		{
			"should fail when login fails",
			fake.NewFakeClientWithScheme(k8scheme.Scheme),
			"blahblah",
			true,
			"secrets \"keycloak-http\" not found",
		},
		{
			"should fail to retrieve user group ID from Keycloak when stdout is empty",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, loginSecret),
			"",
			true,
			"Error retrieving User Group ID from Keycloak",
		},
		{
			"should fail to retrieve user group ID from Keycloak when stdout is incorrect",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, loginSecret),
			"",
			true,
			"Error parsing output returned from Users Group",
		},
		{
			"should fail when Verrazzano secret is not present",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, loginSecret),
			"blahblah'id",
			true,
			"secrets \"verrazzano\" not found",
		},
		{
			"should fail when Verrazzano secret has no password",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, loginSecret, nginxService, &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "verrazzano",
					Namespace: "verrazzano-system",
				},
				Data: map[string][]byte{
					"password": []byte(""),
				},
			}),
			"blahblah'id",
			true,
			"Password field empty in secret",
		},
		{
			"should fail when nginx service is not present",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, loginSecret, &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "verrazzano",
					Namespace: "verrazzano-system",
				},
				Data: map[string][]byte{
					"password": []byte("blah di blah"),
				},
			},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "verrazzano-prom-internal",
						Namespace: "verrazzano-system",
					},
					Data: map[string][]byte{
						"password": []byte("blah di blah"),
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "verrazzano-es-internal",
						Namespace: "verrazzano-system",
					},
					Data: map[string][]byte{
						"password": []byte("blah di blah"),
					},
				}),
			"blahblah'id",
			true,
			"services \"ingress-controller-ingress-nginx-controller\" not found",
		},
		{
			"should pass when able to successfully exec commands on the keycloak pod and all k8s objects are present",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, loginSecret, nginxService, &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "verrazzano",
					Namespace: "verrazzano-system",
				},
				Data: map[string][]byte{
					"password": []byte("blah di blah"),
				},
			},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "verrazzano-prom-internal",
						Namespace: "verrazzano-system",
					},
					Data: map[string][]byte{
						"password": []byte("blah di blah"),
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "verrazzano-es-internal",
						Namespace: "verrazzano-system",
					},
					Data: map[string][]byte{
						"password": []byte("blah di blah"),
					},
				}),
			"blahblah'id",
			false,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.c, testVZ, false)
			execCommand = fakeCreateUserGroupCommand
			if tt.name == "should fail to retrieve user group ID from Keycloak when stdout is empty" {
				execCommand = fakeCreateUserGroupCommandFail
			}
			if tt.name == "should fail to retrieve user group ID from Keycloak when stdout is incorrect" {
				execCommand = fakeCreateUserGroupParseCommandFail
			}
			k8sutilfake.PodSTDOUT = tt.stdout
			err := configureKeycloakRealms(ctx)
			if tt.isErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAppendKeycloakOverrides tests that the Keycloak overrides are generated correctly.
// GIVEN a Verrazzano BOM
// WHEN I call AppendKeycloakOverrides
// THEN the Keycloak overrides Key:Value array has the expected content.
func TestAppendKeycloakOverrides(t *testing.T) {
	assert := assert.New(t)

	const env = "test-env"
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: env,
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestNginxService())

	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendKeycloakOverrides(spi.NewFakeContext(client, vz, false), "", "", "", nil)

	assert.NoError(err, "AppendKeycloakOverrides returned an error")
	assert.Len(kvs, 5, "AppendKeycloakOverrides returned wrong number of Key:Value pairs")

	assert.Contains(kvs, bom.KeyValue{
		Key:       dnsTarget,
		Value:     testKeycloakIngressHost,
		SetString: true,
	})
	assert.Contains(kvs, bom.KeyValue{
		Key:   rulesHost,
		Value: testKeycloakIngressHost,
	})
	assert.Contains(kvs, bom.KeyValue{
		Key:   tlsHosts,
		Value: testKeycloakIngressHost,
	})
	assert.Contains(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: env + "-secret",
	})
}

// TestAppendKeycloakOverridesNoEnvironmentName tests that the Keycloak override for tlsSecret is generated as correctly,
// when the environment name is not defined in the custom resource.
// GIVEN a Verrazzano BOM
// WHEN I call TestAppendKeycloakOverridesNoEnvironmentName
// THEN the Keycloak overrides Key:Value array has the expected value for tlsSecret.
func TestAppendKeycloakOverridesNoEnvironmentName(t *testing.T) {
	assert := assert.New(t)

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: "dev",
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, createTestNginxService())
	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendKeycloakOverrides(spi.NewFakeContext(client, vz, false), "", "", "", nil)

	assert.NoError(err, "AppendKeycloakOverrides returned an error")

	assert.Contains(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: "default-secret",
	})
}

// TestGetEnvironmentName tests that the environment name is returned correctly
// GIVEN a environmentName
// WHEN I call getEnvironmentName
// THEN return the environmentName if it is not empty, else return default.
func TestGetEnvironmentName(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{"", constants.DefaultEnvironmentName},
		{"foobar", "foobar"},
	}

	for _, tt := range tests {
		t.Run(tt.out, func(t *testing.T) {
			assert.Equal(t, tt.out, getEnvironmentName(tt.in))
		})
	}
}

// TestLoginKeycloak tests the login to keycloak interacts with k8s resources as expected
// GIVEN a client
// WHEN I call loginKeycloak
// THEN throw an error if the k8s environment is invalid (bad secret)
func TestLoginKeycloak(t *testing.T) {
	httpSecret := createTestLoginSecret()
	httpSecretEmptyPassword := createTestLoginSecret()
	httpSecretEmptyPassword.Data["password"] = []byte("")
	cfg, restclient, _ := fakeRESTConfig()
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor

	var tests = []struct {
		name  string
		c     client.Client
		isErr bool
	}{
		{
			"should fail when secret does not exist",
			fake.NewFakeClientWithScheme(k8scheme.Scheme),
			true,
		},
		{
			"should fail to find the keycloak password if it is empty",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, httpSecretEmptyPassword),
			true,
		},
		{
			"should log into keycloak when the password is present",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, httpSecret),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sutilfake.PodSTDOUT = "blah"
			err := loginKeycloak(spi.NewFakeContext(tt.c, testVZ, false), cfg, restclient)
			if tt.isErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCreateOrUpdateAuthSecret tests creation of the auth secret
// GIVEN a client
// WHEN I call createOrUpdateAuthSecret
// THEN create the auth secret
func TestCreateOrUpdateAuthSecret(t *testing.T) {
	c := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ctx := spi.NewFakeContext(c, testVZ, false)
	err := createAuthSecret(ctx, "ns", "secret", "user")
	assert.NoError(t, err)
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath)))
}

// TestIsEnabledNilKeycloak tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is nil
//  THEN true is returned
func TestIsEnabledNilKeycloak(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledManagedClusterProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and managed cluster profile
//  THEN false is returned
func TestIsEnabledManagedClusterProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.ManagedCluster
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledProdProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and prod profile
//  THEN false is returned
func TestIsEnabledProdProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.Prod
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledDevProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and dev profile
//  THEN false is returned
func TestIsEnabledDevProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.Dev
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

func getBoolPtr(b bool) *bool {
	return &b
}

// TestClientExists tests that the function returns false/true whether client exists
// GIVEN an array of keycloak Clients
// WHEN I call clientExists
// THEN return true/false whether the client exists in the array of clients
func TestClientExists(t *testing.T) {
	var tests = []struct {
		name string
		in   KeycloakClients
		out  bool
	}{
		{"testEmptyClients",
			KeycloakClients{},
			false,
		},
		{"testClientNotFound",
			KeycloakClients{
				{
					"973973",
					"thisClient",
				},
				{
					"973974",
					"thatClient",
				},
			},
			false,
		},
		{"testClientFound",
			KeycloakClients{
				{
					"973973",
					"thisClient",
				},
				{
					"973974",
					"thatClient",
				},
				{
					"973974",
					"someClient",
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, clientExists(tt.in, "someClient"))
		})
	}
}

// TestUserExists tests that the function returns false/true whether user exists
// GIVEN an array of keycloak Users
// WHEN I call userExists
// THEN return true/false whether the user exists in the array of users
func TestUserExists(t *testing.T) {
	var tests = []struct {
		name string
		in   KeycloakUsers
		out  bool
	}{
		{"testEmptyUsers",
			KeycloakUsers{},
			false,
		},
		{"testUserNotFound",
			KeycloakUsers{
				{
					ID:       "955995",
					Username: "thisUser",
				},
				{
					ID:       "955996",
					Username: "thatUser",
				},
			},
			false,
		},
		{"testUserFound",
			KeycloakUsers{
				{
					ID:       "955995",
					Username: "thisUser",
				},
				{
					ID:       "955996",
					Username: "thatUser",
				},
				{
					ID:       "955997",
					Username: "someUser",
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, userExists(tt.in, "someUser"))
		})
	}
}

// TestRoleExists tests that the function returns false/true whether role exists
// GIVEN an array of keycloak Roles
// WHEN I call roleExists
// THEN return true/false whether the role exists in the array of roles
func TestRoleExists(t *testing.T) {
	var tests = []struct {
		name string
		in   KeycloakRoles
		out  bool
	}{
		{"testEmptyRoles",
			KeycloakRoles{},
			false,
		},
		{"testRoleNotFound",
			KeycloakRoles{
				{
					ID:   "955995",
					Name: "thisRole",
				},
				{
					ID:   "955996",
					Name: "thatRole",
				},
			},
			false,
		},
		{"testRoleFound",
			KeycloakRoles{
				{
					ID:   "955995",
					Name: "thisRole",
				},
				{
					ID:   "955996",
					Name: "thatRole",
				},
				{
					ID:   "955997",
					Name: "someRole",
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, roleExists(tt.in, "someRole"))
		})
	}
}

// TestGroupExists tests that the function returns false/true whether group exists
// GIVEN an array of keycloak Groups
// WHEN I call groupExists
// THEN return true/false whether the group exists in the Keycloak Group structure
func TestGroupExists(t *testing.T) {

	var (
		tests = []struct {
			name string
			in   KeycloakGroups
			out  bool
		}{
			{"testEmptyGroups",
				KeycloakGroups{},
				false,
			},
			{"testGroupNotFound",
				KeycloakGroups{
					{
						ID:        "955995",
						Name:      "thisGroup",
						SubGroups: nil,
					},
					{
						ID:   "",
						Name: "",
						SubGroups: []SubGroup{
							{
								ID:   "333333",
								Name: "subGroup1",
							},
							{
								ID:   "444444",
								Name: "subGroup2",
							},
						},
					},
				},
				false,
			},
			{"testGroupFound",
				KeycloakGroups{
					{
						ID:        "955995",
						Name:      "foundGroup",
						SubGroups: nil,
					},
					{
						ID:   "",
						Name: "",
						SubGroups: []SubGroup{
							{
								ID:   "333333",
								Name: "subGroup1",
							},
							{
								ID:   "444444",
								Name: "subGroup2",
							},
						},
					},
				},
				true,
			},
			{"testSubGroupFound",
				KeycloakGroups{
					{
						ID:        "955995",
						Name:      "someGroup",
						SubGroups: nil,
					},
					{
						ID:   "",
						Name: "",
						SubGroups: []SubGroup{
							{
								ID:   "333333",
								Name: "subGroup1",
							},
							{
								ID:   "444444",
								Name: "subGroup2",
							},
							{
								ID:   "555555",
								Name: "foundGroup",
							},
						},
					},
				},
				true,
			},
		}
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, groupExists(tt.in, "foundGroup"))
		})
	}
}

// TestGetGroupID tests that the function returns the correct groupID for the group name
// GIVEN an array of keycloak Groups
// WHEN I call getGroupID
// THEN return the groupID for the GroupName passed in, empty string for not found
func TestGetGroupID(t *testing.T) {

	var (
		tests = []struct {
			name string
			in   KeycloakGroups
			out  string
		}{
			{"testEmptyGroups",
				KeycloakGroups{},
				"",
			},
			{"testGroupIDNotFound",
				KeycloakGroups{
					{
						ID:        "955995",
						Name:      "thisGroup",
						SubGroups: nil,
					},
					{
						ID:   "",
						Name: "",
						SubGroups: []SubGroup{
							{
								ID:   "333333",
								Name: "subGroup1",
							},
							{
								ID:   "444444",
								Name: "subGroup2",
							},
						},
					},
				},
				"",
			},
			{"testGroupIDFound",
				KeycloakGroups{
					{
						ID:        "999999",
						Name:      "foundGroup",
						SubGroups: nil,
					},
					{
						ID:   "",
						Name: "",
						SubGroups: []SubGroup{
							{
								ID:   "333333",
								Name: "subGroup1",
							},
							{
								ID:   "444444",
								Name: "subGroup2",
							},
						},
					},
				},
				"999999",
			},
			{"testSubGroupIDFound",
				KeycloakGroups{
					{
						ID:        "955995",
						Name:      "someGroup",
						SubGroups: nil,
					},
					{
						ID:   "",
						Name: "",
						SubGroups: []SubGroup{
							{
								ID:   "333333",
								Name: "subGroup1",
							},
							{
								ID:   "444444",
								Name: "subGroup2",
							},
							{
								ID:   "999999",
								Name: "foundGroup",
							},
						},
					},
				},
				"999999",
			},
		}
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, getGroupID(tt.in, "foundGroup"))
		})
	}
}

func TestUpdateKeycloakIngress(t *testing.T) {
	ingress := &networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "keycloak", Namespace: "keycloak"},
	}
	annotations := make(map[string]string)
	annotations["cdd"] = "foo"
	annotations["bar"] = "baz"
	ingress.SetAnnotations(annotations)
	c := fake.NewFakeClientWithScheme(k8scheme.Scheme, ingress, createTestNginxService())
	ctx := spi.NewFakeContext(c, testVZ, false)
	err := updateKeycloakIngress(ctx)
	assert.NoError(t, err)
}
