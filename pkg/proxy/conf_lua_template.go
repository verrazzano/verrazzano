// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

// OidcConfLuaFilename defines the conf.lua file name in OIDC proxy ConfigMap
const OidcConfLuaFilename = "conf.lua"

// OidcConfLuaFileTemplate is the template of conf.lua file in OIDC proxy ConfigMap
const OidcConfLuaFileTemplate = `|
    local ingressUri = 'https://'..'{{ .Ingress }}'
    local oidcProviderHost = "{{ .OidcProviderHost }}"
    local oidcProviderHostInCluster = "{{ .OidcProviderHostInCluster }}"
    local realm ="{{ .OidcRealm }}"
    local callbackPath = "{{ .OidcCallbackPath }}"
    local logoutCallbackPath = "{{ .OidcLogoutCallbackPath }}"
    local oidcClient = "{{ .PKCEClientID }}"
    local oidcDirectAccessClient = "{{ .PGClientID }}"
    -- TODO: fix this -- shouldn't have cookie key in configmap
    local cookieKey = "{{ .RandomString }}"
    local requiredRole = "{{ .RequiredRealmRole }}"
    local authStateTtlInSec = {{ .AuthnStateTTL }}

    local oidcProviderUri = ""
    local oidcProviderInClusterUri = ""
{{- if eq .Mode "oauth-proxy" }}
    oidcProviderUri = 'https://'..oidcProviderHost..'/auth/realms/'..realm
    if oidcProviderHostInCluster and oidcProviderHostInCluster ~= "" then
        oidcProviderInClusterUri = 'http://'..oidcProviderHostInCluster..'/auth/realms/'..realm
    end
{{- else if .Mode "api-proxy" }}
    local oidcProviderUri = read_file("/api-config/keycloak-url");
    if oidcProviderUri then
        oidcProviderUri = oidcProviderUri..'/auth/realms/'..realm
    else
        oidcProviderUri = ""
        me.logJson(ngx.INFO, "No keycloak-url specified in api-config. Using in-cluster keycloak url.")
    end
    local oidcProviderInClusterUri = 'http://keycloak-http.keycloak.svc.cluster.local'..'/auth/realms/'..realm
{{- end }}

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
        cookieKey = cookieKey
    })

    ngx.header["Access-Control-Allow-Headers"] = "authorization"

    if ngx.req.get_method() == "OPTIONS" then
        ngx.status = 200
        ngx.exit(ngx.HTTP_OK)
    end

    auth.info("Checking for authorization header")
    local authHeader = ngx.req.get_headers()["authorization"]
    local token = nil
    if authHeader then
{{- if eq .Mode "api-proxy" }}
        -- console sent access token to api proxy with k8s api request (not cached)
        token = auth.handleBearerToken(authHeader)
{{- else if eq .Mode "oauth-proxy" }}
        -- vz component calling vmi component using basic auth (cached locally)
        token = handleBasicAuth(authHeader)
{{- end }}
        if not token then
            auth.info("No recognized credentials in authorization header")
        fi
    else
        auth.info("No authorization header found")
{{- if eq .Mode "oauth-proxy" }}
        if string.find(ngx.var.request_uri, callbackPath) then
            -- we initiated authentication via pkce, and OP is delivering the code
            -- will redirect to target url, where token will be found in cookie
            auth.oidcHandleCallback()
        elseif string.find(ngx.var.request_uri, logoutCallbackPath) then
            -- logout was triggered, and OP (always?) is calling our logout URL
            auth.oidcHandleLogoutCallback()
        end
        -- still no token, check if caller has a valid token in session (cookie)
        -- redirect after handling callback should end up here
        token = auth.getTokenFromSession()
        if not token then
            -- no token, redirect to OP to authenticate
            auth.oidcAuthenticate()
        end
{{- end }}
    end

    if not token
        auth.unauthorized("Not authenticated")
    end

    -- token will be an id token except when console calls api proxy, then it's an access token
    -- TODO: need to fix this so token handling/consumption is aligned across all clients/backends
    if not auth.isAuthorized(token) then
        auth.forbidden("Not authorized")
    end

{{- if eq .Mode "api-proxy" }}
    -- impersonate the user
    impersonateKubernetesUser(token)
{{- else if eq .Mode "oauth-proxy" }}
    ngx.var.oidc_user = auth.usernameFromIdToken(idToken)
{{- end }}

`
