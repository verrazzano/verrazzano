// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"bytes"
	"text/template"
)

type OidcProxyConfig struct {
	// configmap metadata
	Name      string
	Namespace string
	// for startup.sh
	StartupDir string
	// for reload.sh (none current)
	// for nginx.conf
	SSLVerifyOptions    string
	SSLTrustedCAOptions string
	Host                string
	Port                int
	// for conf.lua
	Ingress                   string
	OIDCProviderHost          string
	OIDCProviderHostInCluster string
	Realm                     string
	OIDCCallbackPath          string
	OIDCLogoutCallbackPath    string
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

func GetOidcProxyConfigMapData(config OidcProxyConfig) (map[string]string, error) {
	// set default values
	config.StartupDir = "`dirname $0`"
	config.Realm = OidcRealmName
	config.PKCEClientID = OidcPkceClientID
	config.PGClientID = OidcPgClientID
	config.OIDCCallbackPath = OidcCallbackPath
	config.OIDCLogoutCallbackPath = OidcLogoutCallbackPath
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
