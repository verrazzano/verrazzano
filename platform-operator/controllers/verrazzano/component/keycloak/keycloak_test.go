// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath         = "../../testdata/test_bom.json"
	testKeycloakIngressHost = "keycloak.test-env.192.132.111.122.nip.io"
	profilesRelativePath    = "../../../../manifests/profiles"
)

var testVZ = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
		Components: vzapi.ComponentSpec{
			Keycloak: &vzapi.KeycloakComponent{
				MySQL: vzapi.MySQLComponent{},
			},
		},
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

func createTestKeycloakAuthConfig() unstructured.Unstructured {
	authConfig := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	authConfig.SetGroupVersionKind(common.GVKAuthConfig)
	authConfig.SetName(common.AuthConfigKeycloak)
	return authConfig
}

func fakeConfigureRealmCommands(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		command := commands[2]
		if strings.Contains(command, "create groups") {
			return "Created new group with id 'quick-brown-fox'", "", nil
		}

		if strings.Contains(command, "get clients") {
			return "[{\"id\" : \"quick-fox\",\"clientId\" : \"jump-window\"}]", "", nil
		}

		if strings.Contains(command, "create clients/") {
			return "Created client secret blah", "", nil
		}

		if strings.Contains(command, "create clients") {
			return "Created client 'blahblah'", "", nil
		}

	}

	return "", "", nil
}

func fakeConfigureRealmCommandsUpdateKeycloakURIFailed(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "create groups") {
			return "Created new group with id 'quick-brown-fox'", "", nil
		}
	}

	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "get clients") {
			return "[{\"id\" : \"quick-fox\",\"clientId\" : \"rancher\"},{\"id\" : \"quick-fox-1\",\"clientId\" : \"verrazzano-pg\"}]", "", nil
		}
	}

	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "update clients") {
			return "", "", fmt.Errorf("failed")
		}
	}
	return "", "", nil
}

func fakeCreateUserGroupCommand(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "create groups") {
			return "Created new group with id 'quick-brown-fox'", "", nil
		}
	}
	return "", "", nil
}

func fakeCreateUserGroupCommandFail(url *url.URL) (string, string, error) {
	return "", "", nil
}

func fakeCreateUserGroupParseCommandFail(url *url.URL) (string, string, error) {
	return "invalidoutput", "", nil
}

func fakeGetRancherClientSecretFromKeycloak(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "client-secret") {
			return "{\"type\":\"secret\",\"value\":\"abcdef\"}", "", nil
		}

		if strings.Contains(commands[2], "get clients") {
			return "[{\"id\" : \"quick-fox\",\"clientId\" : \"rancher\"}]", "", nil
		}
	}

	return "", "", nil
}

func fakeGetRancherClientSecretFromKeycloakGetClientsFails(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "get clients") {
			return "", "", fmt.Errorf("failed")
		}

	}

	return "", "", nil
}

func fakeGetRancherClientSecretFromKeycloakNoRancherClient(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "get clients") {
			return "[{\"id\" : \"quick-fox\",\"clientId\" : \"norancher\"}]", "", nil
		}
	}

	return "", "", nil
}

func fakeGetRancherClientSecretFromKeycloakClientSecretFailed(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "client-secret") {
			return "", "", fmt.Errorf("failed")
		}

		if strings.Contains(commands[2], "get clients") {
			return "[{\"id\" : \"quick-fox\",\"clientId\" : \"rancher\"}]", "", nil
		}
	}

	return "", "", nil
}

func fakeGetRancherClientSecretFromKeycloakClientSecretResultEmpty(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "client-secret") {
			return "", "", nil
		}

		if strings.Contains(commands[2], "get clients") {
			return "[{\"id\" : \"quick-fox\",\"clientId\" : \"rancher\"}]", "", nil
		}
	}

	return "", "", nil
}

func fakeGetRancherClientSecretFromKeycloakClientSecretResultInvalid(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "client-secret") {
			return "invalid", "", nil
		}

		if strings.Contains(commands[2], "get clients") {
			return "[{\"id\" : \"quick-fox\",\"clientId\" : \"rancher\"}]", "", nil
		}
	}

	return "", "", nil
}

func fakeGetRancherClientSecretFromKeycloakClientSecretEmpty(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "client-secret") {
			return "{\"type\":\"secret\",\"value\":\"\"}", "", nil
		}

		if strings.Contains(commands[2], "get clients") {
			return "[{\"id\" : \"quick-fox\",\"clientId\" : \"rancher\"}]", "", nil
		}
	}

	return "", "", nil
}

func fakeGetVerrazzanoUserFromKeycloak(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "get users") {
			return "[{\"id\" : \"quick-fox\",\"username\" : \"verrazzano\"}]", "", nil
		}
	}

	return "", "", nil
}

func fakeGetVerrazzanoUserFromKeycloakFails(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "get users") {
			return "", "", fmt.Errorf("failed")
		}

	}

	return "", "", nil
}

func fakeGetVerrazzanoUserFromKeycloakResultEmpty(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "get users") {
			return "", "", nil
		}
	}

	return "", "", nil
}

func fakeGetVerrazzanoUserFromKeycloakResultInvalid(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "get users") {
			return "invalid", "", nil
		}
	}

	return "", "", nil
}

func fakeGetVerrazzanoUserFromKeycloakNoVerrazzanoUser(url *url.URL) (string, string, error) {
	var commands []string
	if commands = url.Query()["command"]; len(commands) == 3 {
		if strings.Contains(commands[2], "get users") {
			return "[{\"id\" : \"quick-fox\",\"username\" : \"notverrazzano\"}]", "", nil
		}
	}

	return "", "", nil
}

func TestUpdateKeycloakURIs(t *testing.T) {
	k8sutil.ClientConfig = fakeRESTConfig
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	cfg, cli, _ := fakeRESTConfig()
	clientID := "client"
	uriTemplate := "\"redirectUris\": [\"https://client.{{.DNSSubDomain}}/verify-auth\"]"
	tests := []struct {
		name        string
		ctx         spi.ComponentContext
		clientID    string
		uriTemplate string
		wantErr     bool
	}{
		{
			name: "testUpdateKeycloakURIs",
			ctx: spi.NewFakeContext(
				fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(createTestLoginSecret(), createTestNginxService()).Build(),
				testVZ,
				false),
			clientID:    clientID,
			uriTemplate: uriTemplate,
			wantErr:     false,
		},
		{
			name: "testFailForInvalidUriTemplate",
			ctx: spi.NewFakeContext(
				fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(createTestLoginSecret(), createTestNginxService()).Build(),
				testVZ,
				false),
			clientID:    clientID,
			uriTemplate: "test.{{{.DNSSubDomain}}",
			wantErr:     true,
		},
		{
			name: "testFailForNoIngress",
			ctx: spi.NewFakeContext(
				fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(createTestLoginSecret()).Build(),
				testVZ,
				false),
			wantErr:     true,
			clientID:    clientID,
			uriTemplate: uriTemplate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podExecResult := k8sutilfake.PodExecResult
			if tt.wantErr && tt.name == "testFailForNoKeycloakClientsReturned" {
				k8sutilfake.PodExecResult = func(url *url.URL) (string, string, error) {
					return "", "[]", nil
				}
			}
			defer func() { k8sutilfake.PodExecResult = podExecResult }()
			if err := updateKeycloakUris(tt.ctx, cfg, cli, keycloakPod(), tt.clientID, tt.uriTemplate); (err != nil) != tt.wantErr {
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
	podExecFunc := k8sutilfake.PodExecResult
	authConfig := createTestKeycloakAuthConfig()

	keycloakPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakPodName,
			Namespace: ComponentNamespace,
		},
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	var tests = []struct {
		name        string
		c           client.Client
		stdout      string
		isErr       bool
		errContains string
		execFunc    func(url *url.URL) (string, string, error)
	}{
		{
			"should fail when login fails",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(keycloakPod).Build(),
			"blahblah",
			true,
			"secrets \"keycloak-http\" not found",
			fakeCreateUserGroupCommand,
		},
		{
			"should fail to retrieve user group ID from Keycloak when stdout is empty",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			"",
			true,
			"Component Keycloak failed; verrazzano-users group ID from Keycloak is zero length",
			fakeCreateUserGroupCommandFail,
		},
		{
			"should fail to retrieve user group ID from Keycloak when stdout is incorrect",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			"",
			true,
			"failed parsing output returned from verrazzano-users Group",
			fakeCreateUserGroupParseCommandFail,
		},
		{
			"should fail when Verrazzano secret is not present",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			"blahblah'id",
			true,
			"secrets \"verrazzano\" not found",
			fakeCreateUserGroupCommand,
		},
		{
			"should fail when Verrazzano secret has no password",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, nginxService, keycloakPod,
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "verrazzano",
						Namespace: "verrazzano-system",
					},
					Data: map[string][]byte{
						"password": []byte(""),
					},
				}).Build(),
			"blahblah'id",
			true,
			"password field empty in secret",
			fakeCreateUserGroupCommand,
		},
		{
			"should fail when nginx service is not present",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod,
				&v1.Secret{
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
				}).Build(),
			"blahblah'id",
			true,
			"services \"ingress-controller-ingress-nginx-controller\" not found",
			fakeConfigureRealmCommands,
		},
		{
			"fails during updateKeycloakURIs",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, nginxService, keycloakPod,
				&v1.Secret{
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
				}).Build(),
			"blahblah'id",
			true,
			"keycloak/keycloak-0: failed",
			fakeConfigureRealmCommandsUpdateKeycloakURIFailed,
		},
		{
			"should pass when able to successfully exec commands on the keycloak pod and all k8s objects are present",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, nginxService, keycloakPod, &authConfig,
				&v1.Secret{
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
				}).Build(),
			"blahblah'id",
			false,
			"",
			fakeConfigureRealmCommands,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.c, testVZ, false)
			k8sutilfake.PodExecResult = tt.execFunc
			defer func() { k8sutilfake.PodExecResult = podExecFunc }()
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
	a := assert.New(t)

	const env = "test-env"
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: env,
		},
	}

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(createTestNginxService()).Build()

	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendKeycloakOverrides(spi.NewFakeContext(c, vz, false), "", "", "", nil)

	a.NoError(err, "AppendKeycloakOverrides returned an error")
	a.Len(kvs, 7, "AppendKeycloakOverrides returned wrong number of Key:Value pairs")

	a.Contains(kvs, bom.KeyValue{
		Key:       dnsTarget,
		Value:     testKeycloakIngressHost,
		SetString: true,
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   rulesHost,
		Value: testKeycloakIngressHost,
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   tlsHosts,
		Value: testKeycloakIngressHost,
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: keycloakCertificateName,
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   kcIngressClassKey,
		Value: "verrazzano-nginx",
	})
	a.Contains(kvs, bom.KeyValue{
		Key:   dbHostKey,
		Value: "mysql-instances",
	})
}

// TestAppendKeycloakOverridesNoEnvironmentName tests that the Keycloak override for tlsSecret is generated as correctly,
// when the environment name is not defined in the custom resource.
// GIVEN a Verrazzano BOM
// WHEN I call TestAppendKeycloakOverridesNoEnvironmentName
// THEN the Keycloak overrides Key:Value array has the expected value for tlsSecret.
func TestAppendKeycloakOverridesNoEnvironmentName(t *testing.T) {
	a := assert.New(t)

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: "dev",
		},
	}

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(createTestNginxService()).Build()
	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendKeycloakOverrides(spi.NewFakeContext(c, vz, false), "", "", "", nil)

	a.NoError(err, "AppendKeycloakOverrides returned an error")

	a.Contains(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: keycloakCertificateName,
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
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build(),
			true,
		},
		{
			"should fail to find the keycloak password if it is empty",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(httpSecretEmptyPassword).Build(),
			true,
		},
		{
			"should log into keycloak when the password is present",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(httpSecret).Build(),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(c, testVZ, false)
	err := createAuthSecret(ctx, "ns", "secret", "user")
	assert.NoError(t, err)
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilKeycloak tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is nil
//  THEN true is returned
func TestIsEnabledNilKeycloak(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledManagedClusterProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and managed cluster profile
//  THEN false is returned
func TestIsEnabledManagedClusterProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.ManagedCluster
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledProdProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and prod profile
//  THEN false is returned
func TestIsEnabledProdProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.Prod
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledDevProfile tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Keycloak enabled flag is nil and dev profile
//  THEN false is returned
func TestIsEnabledDevProfile(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Keycloak = nil
	cr.Spec.Profile = vzapi.Dev
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

func getBoolPtr(b bool) *bool {
	return &b
}

// TestGetClientId tests that the function returns the Id whether client exists
// GIVEN an array of keycloak Clients
// WHEN I call getClientId
// THEN return the Id of the client if the client exists in the array of clients
func TestGetClientId(t *testing.T) {
	var tests = []struct {
		name string
		in   KeycloakClients
		out  string
	}{
		{"testEmptyClients",
			KeycloakClients{},
			"",
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
			"",
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
			"973974",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, getClientID(tt.in, "someClient"))
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
		in   []KeycloakUser
		out  bool
	}{
		{"testEmptyUsers",
			[]KeycloakUser{},
			false,
		},
		{"testUserNotFound",
			[]KeycloakUser{
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
			[]KeycloakUser{
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
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(ingress, createTestNginxService()).Build()
	ctx := spi.NewFakeContext(c, testVZ, false)
	err := updateKeycloakIngress(ctx)
	assert.NoError(t, err)
}

func TestIsKeycloakReady(t *testing.T) {
	readySecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakCertificateName,
			Namespace: ComponentNamespace,
		},
	}
	scheme := k8scheme.Scheme
	_ = certmanager.AddToScheme(scheme)
	var tests = []struct {
		name    string
		c       client.Client
		isReady bool
	}{
		{
			"should not be ready when certificate not found",
			fake.NewClientBuilder().WithScheme(scheme).Build(),
			false,
		},
		{
			"should not be ready when certificate has no status",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(&certmanager.Certificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keycloakCertificateName,
					Namespace: ComponentNamespace,
				},
			}).Build(),
			false,
		},
		{
			"should not be ready when secret does not exists",
			fake.NewClientBuilder().WithScheme(scheme).Build(),
			false,
		},
		{
			"should not be ready when certificate status is ready but statefulset is not ready",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(readySecret).Build(),
			false,
		},
		{
			"should be ready when certificate status is ready and statefulset is ready",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(readySecret,
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      ComponentName,
						Labels:    map[string]string{"app": "test"},
					},
					Spec: appsv1.StatefulSetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test"},
						},
					},
					Status: appsv1.StatefulSetStatus{
						ReadyReplicas:   1,
						UpdatedReplicas: 1,
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      ComponentName + "-0",
						Labels: map[string]string{
							"app":                      "test",
							"controller-revision-hash": "test-95d8c5d96",
						},
					},
				},
				&appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-95d8c5d96",
						Namespace: ComponentNamespace,
					},
					Revision: 1,
				},
			).Build(),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.c, testVZ, false)
			assert.Equal(t, tt.isReady, isKeycloakReady(ctx))
		})
	}
}

func TestUpgradeStatefulSet(t *testing.T) {
	replicaCount := int32(1)
	enabled := true

	// Initial state of the Keycloak StatefulSet
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicaCount,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: v1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{"app.kubernetes.io/instance": "keycloak", "app.kubernetes.io/name": "keycloak"},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	scheme := k8scheme.Scheme
	_ = certmanager.AddToScheme(scheme)
	var tests = []struct {
		name                 string
		c                    client.Client
		vz                   *vzapi.Verrazzano
		profilesDir          string
		expectedReplicaCount int32
	}{
		{
			"no change to StatefulSet when no affinity overrides",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(statefulSet).Build(),
			&vzapi.Verrazzano{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ComponentName,
					Namespace: ComponentNamespace,
				},
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &enabled,
						},
					},
				},
			},
			"",
			int32(1),
		},
		{
			"no change to StatefulSet when affinity override same as existing definition",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(statefulSet).Build(),
			&vzapi.Verrazzano{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ComponentName,
					Namespace: ComponentNamespace,
				},
			},
			profilesRelativePath,
			int32(1),
		},
		{
			"StatefulSet replica count scaled to 0 when ValueOverrides not same as StatefulSet",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(statefulSet).Build(),
			&vzapi.Verrazzano{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ComponentName,
					Namespace: ComponentNamespace,
				},
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							InstallOverrides: vzapi.InstallOverrides{
								ValueOverrides: []vzapi.Overrides{
									{
										Values: &apiextensionsv1.JSON{
											Raw: []byte("{\"affinity\": \"podAntiAffinity:\\n  preferredDuringSchedulingIgnoredDuringExecution:\\n    - weight: 100\\n\"}"),
										},
									},
								},
							},
						},
					},
				},
			},
			"",
			int32(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx spi.ComponentContext
			if len(tt.profilesDir) > 0 {
				ctx = spi.NewFakeContext(tt.c, tt.vz, false, tt.profilesDir)
			} else {
				ctx = spi.NewFakeContext(tt.c, tt.vz, false)
			}
			err := upgradeStatefulSet(ctx)
			assert.NoError(t, err)

			stsUpdated := appsv1.StatefulSet{}
			err = tt.c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, &stsUpdated)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedReplicaCount, *stsUpdated.Spec.Replicas)
		})
	}
}

// TestGetRancherClientSecretFromKeycloak tests getting rancher client secrets
// GIVEN a client, and a k8s environment
// WHEN I call GetRancherClientSecretFromKeycloak
// THEN returns an rancher client secret, otherwise returning an error if the environment is invalid
func TestGetRancherClientSecretFromKeycloak(t *testing.T) {
	loginSecret := createTestLoginSecret()
	k8sutil.ClientConfig = fakeRESTConfig
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	podExecFunc := k8sutilfake.PodExecResult

	keycloakPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakPodName,
			Namespace: ComponentNamespace,
		},
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	var tests = []struct {
		name        string
		c           client.Client
		isErr       bool
		errContains string
		execFunc    func(url *url.URL) (string, string, error)
	}{
		{
			"should fail when login fails",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(keycloakPod).Build(),
			true,
			"secrets \"keycloak-http\" not found",
			fakeGetRancherClientSecretFromKeycloak,
		},
		{
			"should fail when fails to get clients",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"keycloak/keycloak-0: failed",
			fakeGetRancherClientSecretFromKeycloakGetClientsFails,
		},
		{
			"should not fail when rancher client id does not exist",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			false,
			"",
			fakeGetRancherClientSecretFromKeycloakNoRancherClient,
		},
		{
			"should fail when fetching client secret fails",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"",
			fakeGetRancherClientSecretFromKeycloakClientSecretFailed,
		},
		{
			"should fail when client secret result is empty",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"",
			fakeGetRancherClientSecretFromKeycloakClientSecretResultEmpty,
		},
		{
			"should fail when client secret result is invalid",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"",
			fakeGetRancherClientSecretFromKeycloakClientSecretResultInvalid,
		},
		{
			"should fail when client secret is empty",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"client secret is empty",
			fakeGetRancherClientSecretFromKeycloakClientSecretEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.c, testVZ, false)
			k8sutilfake.PodExecResult = tt.execFunc
			defer func() { k8sutilfake.PodExecResult = podExecFunc }()
			_, err := GetRancherClientSecretFromKeycloak(ctx)
			if tt.isErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetVerrazzanoUserFromKeycloak tests getting verrazzano user
// GIVEN a client, and a k8s environment
// WHEN I call GetVerrazzanoUserFromKeycloak
// THEN returns a verrazzano user struct, otherwise returning an error if the environment is invalid
func TestGetVerrazzanoUserFromKeycloak(t *testing.T) {
	loginSecret := createTestLoginSecret()
	k8sutil.ClientConfig = fakeRESTConfig
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	podExecFunc := k8sutilfake.PodExecResult

	keycloakPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakPodName,
			Namespace: ComponentNamespace,
		},
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	var tests = []struct {
		name        string
		c           client.Client
		isErr       bool
		errContains string
		execFunc    func(url *url.URL) (string, string, error)
	}{
		{
			"should fail when login fails",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(keycloakPod).Build(),
			true,
			"secrets \"keycloak-http\" not found",
			fakeGetVerrazzanoUserFromKeycloak,
		},
		{
			"should fail when fails to get users",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"keycloak/keycloak-0: failed",
			fakeGetVerrazzanoUserFromKeycloakFails,
		},
		{
			"should fail when get users result is empty",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"",
			fakeGetVerrazzanoUserFromKeycloakResultEmpty,
		},
		{
			"should fail when get users result is invalid",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"",
			fakeGetVerrazzanoUserFromKeycloakResultInvalid,
		},
		{
			"should fail when verrazzano user is not found",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(loginSecret, keycloakPod).Build(),
			true,
			"verrazzano user does not exist",
			fakeGetVerrazzanoUserFromKeycloakNoVerrazzanoUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.c, testVZ, false)
			k8sutilfake.PodExecResult = tt.execFunc
			defer func() { k8sutilfake.PodExecResult = podExecFunc }()
			_, err := GetVerrazzanoUserFromKeycloak(ctx)
			if tt.isErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
