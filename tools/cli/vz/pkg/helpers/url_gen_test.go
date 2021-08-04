// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientID(t *testing.T) {
	var (
		tests = []struct {
			args   []string
			output string
		}{
			{
				[]string{"webui"},
				"webui",
			},
			{
				[]string{"fakeclientid"},
				"fakeclientid",
			},
		}
	)

	asserts := assert.New(t)
	for _, test := range tests {
		err := os.Setenv("VZ_CLIENT_ID", test.args[0])
		asserts.NoError(err)
		cliendIDRecv := GetClientID()
		asserts.Equal(cliendIDRecv, test.output)
	}
}

func TestGetVerrazzanoRealm(t *testing.T) {
	var (
		tests = []struct {
			args   []string
			output string
		}{
			{
				[]string{"verrazzano-realm"},
				"verrazzano-realm",
			},
			{
				[]string{"fakeverrazzanosystem"},
				"fakeverrazzanosystem",
			},
		}
	)

	asserts := assert.New(t)
	for _, test := range tests {
		err := os.Setenv("VZ_REALM", test.args[0])
		asserts.NoError(err)
		realmRecv := GetVerrazzanoRealm()
		asserts.Equal(realmRecv, test.output)
	}
}

func TestGetKeycloakURL(t *testing.T) {
	var (
		tests = []struct {
			args   []string
			output string
		}{
			{
				[]string{"https://keycloak.xyz.nio.io"},
				"https://keycloak.xyz.nio.io",
			},
			{
				[]string{"http://localhost:8080/fake"},
				"http://localhost:8080/fake",
			},
		}
	)

	asserts := assert.New(t)
	for _, test := range tests {
		err := os.Setenv("VZ_KEYCLOAK_URL", test.args[0])
		asserts.NoError(err)
		keycloakURLRecv, err := GetKeycloakURL("")
		asserts.NoError(err)
		asserts.Equal(keycloakURLRecv, test.output)
	}
}

func TestGenerateKeycloakTokenURL(t *testing.T) {
	var (
		tests = []struct {
			args   []string
			output string
		}{
			{
				[]string{"https://keycloak.xyz.nio.io", "verrazzano-realm"},
				"https://keycloak.xyz.nio.io/auth/realms/verrazzano-realm/protocol/openid-connect/token",
			},
			{
				[]string{"http://localhost:8080/fake", "fakeverrazzanorealm"},
				"http://localhost:8080/fake/auth/realms/fakeverrazzanorealm/protocol/openid-connect/token",
			},
		}
	)

	asserts := assert.New(t)
	for _, test := range tests {
		err := os.Setenv("VZ_KEYCLOAK_URL", test.args[0])
		asserts.NoError(err)
		err = os.Setenv("VZ_REALM", test.args[1])
		asserts.NoError(err)
		URLRecv, err := GenerateKeycloakTokenURL("")
		asserts.NoError(err)
		asserts.Equal(URLRecv, test.output)
	}
}

func TestGenerateRedirectURI(t *testing.T) {
	asserts := assert.New(t)
	for i := 2; i < 5; i++ {
		listener, err := net.Listen("tcp", ":0")
		asserts.NoError(err)
		expectedURI := "http://localhost:" + strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
		actualURI := GenerateRedirectURI(listener)
		asserts.Equal(expectedURI, actualURI)
	}
}

func TestGenerateKeycloakAPIURL(t *testing.T) {
	asserts := assert.New(t)
	var (
		tests = []struct {
			args   []string
			output string
		}{
			{
				[]string{"https://keycloak.xyz.nio.io", "randomclientid", "verrazzano-realm", "randomstate", "http://localhost:8080", "randomcodechallenge"},
				"https://keycloak.xyz.nio.io/auth/realms/verrazzano-realm/protocol/openid-connect/auth?code_challenge_method=S256&client_id=randomclientid&state=randomstate&redirect_uri=http://localhost:8080&code_challenge=randomcodechallenge&response_type=code",
			},
		}
	)
	for _, test := range tests {
		err := os.Setenv("VZ_KEYCLOAK_URL", test.args[0])
		asserts.NoError(err)
		err = os.Setenv("VZ_REALM", test.args[2])
		asserts.NoError(err)
		err = os.Setenv("VZ_CLIENT_ID", test.args[1])
		asserts.NoError(err)

		URLRecv, err := GenerateKeycloakAPIURL(test.args[5], test.args[4], test.args[3], "")
		asserts.NoError(err)
		uTest, err := url.Parse(test.output)
		asserts.NoError(err)
		uRecv, err := url.Parse(URLRecv)
		asserts.Equal(uTest.Path, uRecv.Path)
		asserts.NoError(err)
		asserts.Equal(uTest.Host, uRecv.Host)
	}
}
