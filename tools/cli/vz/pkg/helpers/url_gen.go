// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Generates redirect_uri using the given port number
// return string of the form `http://localhost:1234`
func GenerateRedirectURI(listener net.Listener) string {
	port := listener.Addr().(*net.TCPAddr).Port
	u := &url.URL{
		Scheme: "http",
		Host:   "localhost:" + strconv.Itoa(port),
	}
	return u.String()
}

// Accepts url parameters in the form of map[string][string]
// Returns concatenated list of url parameters
// Return string of the form `code=xyz&status=abc`
func ConcatURLParams(urlParams map[string]string) string {
	var params []string
	for k, v := range urlParams {
		params = append(params, k+"="+v)
	}
	return strings.Join(params, "&")
}

// Returns the oidc client id
func GetClientId() string{
	clientId := os.Getenv("VZ_CLIENT_ID")
	// Look for the matching environment variable, return default if not found
	return clientId
}

// Returns the keycloak base url
func GetKeycloakURL() string{
	keycloakUrl := os.Getenv("VZ_KEYCLOAK_URL")
	// Look for the matching environment variable, return default if not found
	return keycloakUrl
}

// Returns the realm name the oidc client is part of
func GetVerrazzanoRealm() string{
	realmName := os.Getenv("VZ_REALM")
	// Look for the matching environment variable, return default if not found
	return realmName
}

// Generates the keycloak api url to login
// Return string of the form `https://keycloak.xyz.io:123/auth/realms/verrazzano-system/protocol/openid-connect/auth?redirect_uri=abc&state=xyz...`
func GenerateKeycloakAPIURL(codeChallenge string, redirectUri string) string {
	urlParams := map[string]string{
		"client_id":             GetClientId(),
		"response_type":         "code",
		"state":                 "fj8o3n7bdy1op5",
		"redirect_uri":          redirectUri,
		"code_challenge":        codeChallenge,
		"code_challenge_method": "S256",
	}

	host :=     GetKeycloakURL()
	path :=     "auth/realms/" + GetVerrazzanoRealm() + "/protocol/openid-connect/auth"
	rawQuery := ConcatURLParams(urlParams)

	return host + "/" + path + "?" + rawQuery
}

// Gnerates and returns keycloak server api url to get the jwt token
// Return string of the form `https://keycloak.xyz.io:123/auth/realms/verrazzano-system/protocol/openid-connect/token
func GenerateKeycloakTokenURL() string {

	host :=   GetKeycloakURL()
	path :=   "auth/realms/" + GetVerrazzanoRealm() + "/protocol/openid-connect/token"
	return host + "/" + path
}
