// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package keycloak

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var keycloakClientIds string = "[ {\n  \"id\" : \"a732-249893586af2\",\n  \"clientId\" : \"account\"\n}, {\n  \"id\" : \"4256-a350-e46eb48e8606\",\n  \"clientId\" : \"account-console\"\n}, {\n  \"id\" : \"4c1d-8d1b-68635e005567\",\n  \"clientId\" : \"admin-cli\"\n}, {\n  \"id\" : \"4350-ab70-17c37dd995b9\",\n  \"clientId\" : \"broker\"\n}, {\n  \"id\" : \"4f6d-a495-0e9e3849608e\",\n  \"clientId\" : \"realm-management\"\n}, {\n  \"id\" : \"4d92-9d64-f201698d2b79\",\n  \"clientId\" : \"security-admin-console\"\n}, {\n  \"id\" : \"4160-8593-32697ebf2c11\",\n  \"clientId\" : \"verrazzano-oauth-client\"\n}, {\n  \"id\" : \"bde9-9374bd6a38fd\",\n  \"clientId\" : \"verrazzano-pg\"\n}, {\n  \"id\" : \"8327-13cdbfe3b000\",\n  \"clientId\" : \"verrazzano-pkce\"\n\n}, {\n  \"id\" : \"494a-b7ec-b05681cafc73\",\n  \"clientId\" : \"webui\"\n} ]"

// fakeBash verifies that the correct parameter values are passed to upgrade
func fakeBash(_ ...string) (string, string, error) {
	return "success", "", nil
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	firstArg := os.Args[0]
	cmd := exec.Command(firstArg, cs...)
	//	cmd := exec.Command(os.Args[0], cs...)
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
			args: spi.NewContext(zap.S(),
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCommand = fakeExecCommand
			setBashFunc(fakeBash)
			defer func() { execCommand = exec.Command }()
			if err := updateKeycloakUris(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("updateKeycloakUris() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Future Test Templates
/* func TestKeycloakComponent_PostUpgrade(t *testing.T) {
	type fields struct {
		HelmComponent helm.HelmComponent
	}
	type args struct {
		ctx spi.ComponentContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := KeycloakComponent{
				HelmComponent: tt.fields.HelmComponent,
			}
			if err := c.PostUpgrade(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("PostUpgrade() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
} */

/* func TestNewComponent(t *testing.T) {
	tests := []struct {
		name string
		want spi.Component
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewComponent(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewComponent() = %v, want %v", got, tt.want)
			}
		})
	}
} */
