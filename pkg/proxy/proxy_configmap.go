// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"bytes"
	"text/template"
)

// ProxyModeAPI mode in which twhere the proxy accepts only bearer tokens (from console) and impersonates the token's subject to Kubenetes
const ProxyModeAPI = "api-proxy"

// ProxyModeOauth mode in which the proxy supports both Password Grant and PKCE OIDC flows, and provides authentication/sso for VMI components
const ProxyModeOauth = "oauth-proxy"

// OidcRealmName is the name of the verrazzano system realm in Keycloak
const OidcRealmName = "verrazzano-system"

// OidcPkceClientID is the name of the verrazzano PKCE client
const OidcPkceClientID = "verrazzano-pkce"

// OidcPgClientID is the name of the verrazzano password grant client
const OidcPgClientID = "verrazzano-pg"

// OidcRequiredRealmRole is the role required to access Verrazzano APIs viea the proxy
const OidcRequiredRealmRole = "vz-api-access"

// OidcAuthnStateTTL is the TTL for entries in the basic auth creds cache
const OidcAuthnStateTTL = 300

// OidcCallbackPath is the callback URL path of OIDC authentication redirect
const OidcCallbackPath = "/_authentication_callback"

// OidcLogoutCallbackPath is the callback URL path after logout
const OidcLogoutCallbackPath = "/_logout"

// OidcProxyConfig type represents the config settings for a proxy instance
type OidcProxyConfig struct {
	// proxy mode: api-proxy or oauth-proxy
	Mode string
	// configmap metadata
	Name      string
	Namespace string
	// for reload.sh (none current)
	// for nginx.conf
	Host string
	Port int
	// for conf.lua
	Ingress                   string
	OidcProviderHost          string
	OidcProviderHostInCluster string
	Realm                     string
	OidcCallbackPath          string
	OidcLogoutCallbackPath    string
	PKCEClientID              string
	PGClientID                string
	RandomString              string
	RequiredRealmRole         string
	AuthnStateTTL             int
	// for auth.lua (none current)
}

func oidcStartup(values interface{}) (string, error) {
	return executeTemplateWithValues("proxy_startup", OidcStartupFileTemplate, values)
}

func oidcNginxConf(values interface{}) (string, error) {
	return executeTemplateWithValues("proxy_nginx_conf", OidcNginxConfFileTemplate, values)
}

func oidcConfLua(values interface{}) (string, error) {
	return executeTemplateWithValues("proxy_conf_lua", OidcConfLuaFileTemplate, values)
}

func oidcAuthLua(values interface{}) (string, error) {
	return executeTemplateWithValues("proxy_auth_lua", OidcAuthLuaFileTemplate, values)
}

func executeTemplateWithValues(templateName string, templateString string, values interface{}) (string, error) {
	t, err := template.New(templateName).Parse(templateString)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	err = t.Execute(buf, values)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GetOidcProxyConfigMapData returns a map containing config files for a proxy instance
func GetOidcProxyConfigMapData(config OidcProxyConfig) (map[string]string, error) {
	// set default values
	config.Mode = ProxyModeOauth
	config.Realm = OidcRealmName
	config.PKCEClientID = OidcPkceClientID
	config.PGClientID = OidcPgClientID
	config.OidcCallbackPath = OidcCallbackPath
	config.OidcLogoutCallbackPath = OidcLogoutCallbackPath
	config.RequiredRealmRole = OidcRequiredRealmRole
	config.AuthnStateTTL = OidcAuthnStateTTL
	// execute the templates
	authLua, err := oidcAuthLua(config)
	if err != nil {
		return nil, err
	}
	confLua, err := oidcConfLua(config)
	if err != nil {
		return nil, err
	}
	nginxConf, err := oidcNginxConf(config)
	if err != nil {
		return nil, err
	}
	startup, err := oidcStartup(config)
	if err != nil {
		return nil, err
	}
	// return resulting map
	return map[string]string{
		OidcAuthLuaFilename:   authLua,
		OidcConfLuaFilename:   confLua,
		OidcStartupFilename:   startup,
		OidcNginxConfFilename: nginxConf,
	}, nil
}
