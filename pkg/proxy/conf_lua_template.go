// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

// NOTE: Do not add any constants or other variables to this file. It is used as text input
// to a shell script that generates the verrazzano-api-proxy-configmap.yaml file. That script
// depends on there being exactly two constants defined, each of which has a canonical name format,
// and also depends on the quote characters used and their placement. Do not add backtick characters
// anywhere here, including in comments. Add new constants, variables, and functions to proxy_config.go.

// NOTE: If you change this file, you must regenerate the verrazzano-api-proxy-configmap.yaml file,
// by running "make generate-api-proxy-configmap", and check in the regenerated file if it's different.

// OidcConfLuaFilename defines the conf.lua file name in OIDC proxy ConfigMap
const OidcConfLuaFilename = "conf.lua"

// OidcConfLuaFileTemplate is the template of conf.lua file in OIDC proxy ConfigMap
const OidcConfLuaFileTemplate = `    local ingressUri = 'https://'..'{{ .Ingress }}'
    local callbackPath = "{{ .OidcCallbackPath }}"
    local logoutCallbackPath = "{{ .OidcLogoutCallbackPath }}"

    local auth = require("auth").config({
        ingressUri = ingressUri,
        callbackPath = callbackPath,
        logoutCallbackPath = logoutCallbackPath,
        callbackUri = ingressUri..callbackPath
    })

{{ if eq .Mode "oauth-proxy" }}
    ngx.header["Access-Control-Allow-Headers"] = "authorization"
{{ end }}

    if ngx.req.get_method() == "OPTIONS" then
        ngx.status = 200
        ngx.exit(ngx.HTTP_OK)
    end

    local authHeader = ngx.req.get_headers()["authorization"]
    local token = nil
    if authHeader then
{{ if eq .Mode "api-proxy" }}
        -- console sent access token with k8s api request (not cached)
        token = auth.handleBearerToken(authHeader)
{{ else if eq .Mode "oauth-proxy" }}
        -- vz component calling vmi component using basic auth (cached locally)
        token = auth.handleBasicAuth(authHeader)
{{ end }}
        if not token then
            auth.info("No recognized credentials in authorization header")
        end
    else
{{ if eq .Mode "api-proxy" }}
        auth.info("No authorization header")
{{ else if eq .Mode "oauth-proxy" }}
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
{{ end }}
    end

    if not token then
        auth.unauthorized("Not authenticated")
    end

    -- token will be an id token except when console calls api proxy, then it's an access token
    if not auth.isAuthorized(token) then
        auth.forbidden("Not authorized")
    end

{{ if eq .Mode "api-proxy" }}
    local args = ngx.req.get_uri_args()
    if args.cluster then
        auth.handleExternalAPICall(token)
    else
        auth.handleLocalAPICall(token)
    end
{{ else if eq .Mode "oauth-proxy" }}
    -- set the oidc_user
    ngx.var.oidc_user = auth.usernameFromIdToken(token)
    auth.info("Authorized: oidc_user is "..ngx.var.oidc_user)
{{ end }}
`
