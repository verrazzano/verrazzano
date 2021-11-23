// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	testBomFilePath         = "../../testdata/test_bom.json"
	testKeycloakIngressHost = "keycloak.test-env.192.132.111.122.nip.io"
)

var keycloakClientIds string = "[ {\n  \"id\" : \"a732-249893586af2\",\n  \"clientId\" : \"account\"\n}, {\n  \"id\" : \"4256-a350-e46eb48e8606\",\n  \"clientId\" : \"account-console\"\n}, {\n  \"id\" : \"4c1d-8d1b-68635e005567\",\n  \"clientId\" : \"admin-cli\"\n}, {\n  \"id\" : \"4350-ab70-17c37dd995b9\",\n  \"clientId\" : \"broker\"\n}, {\n  \"id\" : \"4f6d-a495-0e9e3849608e\",\n  \"clientId\" : \"realm-management\"\n}, {\n  \"id\" : \"4d92-9d64-f201698d2b79\",\n  \"clientId\" : \"security-admin-console\"\n}, {\n  \"id\" : \"4160-8593-32697ebf2c11\",\n  \"clientId\" : \"verrazzano-oauth-client\"\n}, {\n  \"id\" : \"bde9-9374bd6a38fd\",\n  \"clientId\" : \"verrazzano-pg\"\n}, {\n  \"id\" : \"8327-13cdbfe3b000\",\n  \"clientId\" : \"verrazzano-pkce\"\n\n}, {\n  \"id\" : \"494a-b7ec-b05681cafc73\",\n  \"clientId\" : \"webui\"\n} ]"
var keycloakErrorClientIds string = "[ {\n  \"id\" : \"a732-249893586af2\",\n  \"clientId\" : \"account\"\n}, {\n  \"id\" : \"4256-a350-e46eb48e8606\",\n  \"clientId\" : \"account-console\"\n}, {\n  \"id\" : \"4c1d-8d1b-68635e005567\",\n  \"clientId\" : \"admin-cli\"\n}, {\n  \"id\" : \"4f6d-a495-0e9e3849608e\",\n  \"clientId\" : \"realm-management\"\n}, {\n  \"id\" : \"4d92-9d64-f201698d2b79\",\n  \"clientId\" : \"security-admin-console\"\n}, {\n  \"id\" : \"4160-8593-32697ebf2c11\",\n  \"clientId\" : \"verrazzano-oauth-client\"\n}, {\n  \"id\" : \"bde9-9374bd6a38fd\",\n  \"clientId\" : \"verrazzano-pg\"\n}, {\n  \"id\" : \"494a-b7ec-b05681cafc73\",\n  \"clientId\" : \"webui\"\n} ]"

// fakeBash mocks a successful script run
func fakeBash(_ ...string) (string, string, error) {
	return "success", "", nil
}

// fakeBashFail mocks a failed script run
func fakeBashFail(_ ...string) (string, string, error) {
	return "fail", "Script Failed", errors.New("Script Failed")
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

func Test_updateKeycloakUris(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: "dev",
		},
	}

	tests := []struct {
		name    string
		args    spi.ComponentContext
		wantErr bool
	}{
		{
			name: "testUpdateKeycloakURIs",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-http",
						Namespace: "keycloak",
					},
					Data: map[string][]byte{"password": []byte("password")},
				},
					&v1.Service{
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
					}),
				vz,
				false),
			wantErr: false,
		},
		{
			name: "testFailForNoKeycloakSecret",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme,
					&v1.Service{
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
					}),
				vz,
				false),
			wantErr: true,
		},
		{
			name: "testFailForKeycloakSecretPasswordEmpty",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-http",
						Namespace: "keycloak",
					},
					Data: map[string][]byte{"password": []byte("")},
				},
					&v1.Service{
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
					}),
				vz,
				false),
			wantErr: true,
		},
		{
			name: "testFailForAuthenticationToKeycloak",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-http",
						Namespace: "keycloak",
					},
					Data: map[string][]byte{"password": []byte("password")},
				},
					&v1.Service{
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
					}),
				vz,
				false),
			wantErr: true,
		},
		{
			name: "testFailForNoKeycloakClientsReturned",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-http",
						Namespace: "keycloak",
					},
					Data: map[string][]byte{"password": []byte("password")},
				},
					&v1.Service{
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
					}),
				vz,
				false),
			wantErr: true,
		},
		{
			name: "testFailForKeycloakUserNotFound",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-http",
						Namespace: "keycloak",
					},
					Data: map[string][]byte{"password": []byte("password")},
				},
					&v1.Service{
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
					}),
				vz,
				false),
			wantErr: true,
		},
		{
			name: "testFailForNoIngress",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-http",
						Namespace: "keycloak",
					},
					Data: map[string][]byte{"password": []byte("password")},
				}),
				vz,
				false),
			wantErr: true,
		},
		{
			name: "testScriptFailure",
			args: spi.NewFakeContext(
				fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "keycloak-http",
						Namespace: "keycloak",
					},
					Data: map[string][]byte{"password": []byte("password")},
				},
					&v1.Service{
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
					}),
				vz,
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

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Service{
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
	})

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

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &v1.Service{
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
	})
	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendKeycloakOverrides(spi.NewFakeContext(client, vz, false), "", "", "", nil)

	assert.NoError(err, "AppendKeycloakOverrides returned an error")

	assert.Contains(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: "default-secret",
	})
}
