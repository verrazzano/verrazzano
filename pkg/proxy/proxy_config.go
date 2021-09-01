// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"bytes"
	"text/template"
)

// OidcRealmName is the name of the verrazzano system realm in Keycloak
const OidcRealmName = "verrazzano-system"

// OidcPkceClientID is the name of the verrazzano PKCE client
const OidcPkceClientID = "verrazzano-pkce"

// OidcPgClientID is the name of the verrazzano password grant client
const OidcPgClientID = "verrazzano-pg"

// OidcRequiredRealmRole is the role required to access Verrazzano APIs viea the proxy
const OidcRequiredRealmRole = "vz_api_access"

// OidcAuthnStateTTL is the TTL for entries in the basic auth creds cache
const OidcAuthnStateTTL = 300

// OidcCallbackPath is the callback URL path of OIDC authentication redirect
const OidcCallbackPath = "/_authentication_callback"

// OidcLogoutCallbackPath is the callback URL path after logout
const OidcLogoutCallbackPath = "/_logout"

// OidcProxyConfig type represents the config settings for a proxy instance
type OidcProxyConfig struct {
	// oidc realm/client config and required role
	OidcRealm         string
	PKCEClientID      string
	PGClientID        string
	RequiredRealmRole string

	// callback paths
	OidcCallbackPath       string
	OidcLogoutCallbackPath string

	// oidc provider hosts
	OidcProviderHost          string
	OidcProviderHostInCluster string

	// ttl for entries in basic auth cache
	AuthnStateTTL int
}

func proxyStartup(values interface{}) (string, error) {
	return executeTemplateWithValues("proxy_startup", OidcStartupFileTemplate, values)
}

func proxyReload(values interface{}) (string, error) {
	return executeTemplateWithValues("proxy_reload", OidcReloadFileTemplate, values)
}

func proxyNginxConf(values interface{}) (string, error) {
	return executeTemplateWithValues("proxy_nginx_conf", OidcNginxConfFileTemplate, values)
}

func proxyConfLua(values interface{}) (string, error) {
	return executeTemplateWithValues("proxy_conf_lua", OidcConfLuaFileTemplate, values)
}

func proxyAuthLua(values interface{}) (string, error) {
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

// GetProxyConfigmapData returns a map containing config files for a proxy instance
func GetProxyConfigmapData(config OidcProxyConfig) (map[string]string, error) {
	// execute the templates
	confLua, err := proxyConfLua(config)
	if err != nil {
		return nil, err
	}
	authLua, err := proxyAuthLua(config)
	if err != nil {
		return nil, err
	}
	nginxConf, err := proxyNginxConf(config)
	if err != nil {
		return nil, err
	}
	startup, err := proxyStartup(config)
	if err != nil {
		return nil, err
	}
	reload, err := proxyReload(config)
	if err != nil {
		return nil, err
	}

	cm := map[string]string{
		OidcConfLuaFilename:   confLua,
		OidcAuthLuaFilename:   authLua,
		OidcNginxConfFilename: nginxConf,
		OidcStartupFilename:   startup,
		OidcReloadFilename:    reload,
	}

	return cm, nil
}
