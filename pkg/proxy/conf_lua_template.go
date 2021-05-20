// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

// OidcCallbackPath is the callback URL path of OIDC authentication redirect
const OidcCallbackPath = "/_authentication_callback"

// OidcLogoutCallbackPath is the callback URL path after logout
const OidcLogoutCallbackPath = "/_logout"

// OidcConfLua defines the conf.lua file name in OIDC proxy ConfigMap
const OidcConfLuaFilename = "conf.lua"

// OidcConfLuaTemp is the template of conf.lua file in OIDC proxy ConfigMap
const OidcConfLuaFileTemplate = `|
    local ingressUri = 'https://'..'{{ .Ingress }}'
    local oidcProviderHost = "{{ .OIDCProviderHost }}"
    local oidcProviderHostInCluster = "{{ .OIDCProviderHostInCluster }}"
    local realm ="{{ .Realm }}"
    local callbackPath = "{{ .OIDCCallbackPath }}"
    local logoutCallbackPath = "{{ .OIDCLogoutCallbackPath }}"
    local oidcClient = "{{ .PKCEClientID }}"
    local oidcDirectAccessClient = "{{ .PGClientID }}"
    local cookieKey = "{{ .RandomString }}"
    local requiredRole = "{{ .RequiredRealmRole }}"
    local authStateTtlInSec = {{ .AuthnStateTTL }}
    local bearerServiceAccountToken = false

    local oidcProviderInClusterUri = ""
    if oidcProviderHostInCluster and oidcProviderHostInCluster ~= "" then
        oidcProviderInClusterUri = 'http://'..oidcProviderHostInCluster..'/auth/realms/'..realm
    end
    local oidcProviderUri = 'https://'..oidcProviderHost.. '/auth/realms/'..realm

    local auth = require("auth").config({
        ingressUri = ingressUri,
        callbackPath = callbackPath,
        callbackUri = ingressUri..callbackPath,
        logoutCallbackUri = ingressUri..logoutCallbackPath,
        oidcProviderUri = oidcProviderUri,
        oidcProviderAuthUri = oidcProviderUri..'/protocol/openid-connect/auth',
        oidcProviderInClusterUri = oidcProviderInClusterUri,
        oidcClient = oidcClient,
        oidcDirectAccessClient = oidcDirectAccessClient,
        authStateTtlInSec = authStateTtlInSec,
        cookieKey = cookieKey,
        bearerServiceAccountToken = bearerServiceAccountToken
    })
    ngx.header["Access-Control-Allow-Origin"] =  ngx.req.get_headers()["origin"]
    ngx.header["Access-Control-Allow-Headers"] =  "authorization"
    if ngx.req.get_method() == "OPTIONS" then
        ngx.status = 200
        ngx.exit(ngx.HTTP_OK)
    end
    auth.info("Extracting authorization header from request.")
    local authHeader = ngx.req.get_headers()["authorization"]
    if not authHeader then
        if string.find(ngx.var.request_uri, callbackPath) then
            auth.handleCallback()
        elseif string.find(ngx.var.request_uri, logoutCallbackPath) then
            auth.handleLogoutCallback()
        end
        local ck = auth.authenticated()
        if ck then
            if auth.hasRequiredRole(ck.it, requiredRole) then
                ngx.var.oidc_user = auth.usernameFromIdToken(ck.it)
            else
                auth.logout()
                return
            end
        else
            auth.authenticate()
        end
    else
        local idToken = auth.authHeader(authHeader)
        if idToken then
            if auth.hasRequiredRole(idToken, requiredRole) then
                ngx.var.oidc_user = auth.usernameFromIdToken(idToken)
            else
                auth.forbidden("User does not have a required realm role")
            end
        end
    end
`
