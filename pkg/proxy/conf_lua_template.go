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
const OidcConfLuaFileTemplate = `local ingressHost = ngx.req.get_headers()["x-forwarded-host"]
    local ingressUri = 'https://'..ingressHost
    local callbackPath = "{{ .OidcCallbackPath }}"
    local logoutCallbackPath = "{{ .OidcLogoutCallbackPath }}"

    local auth = require("auth").config({
        callbackUri = ingressUri..callbackPath,
        logoutCallbackUri = ingressUri..logoutCallbackPath
    })

    -- TODO: is this needed here? Is it better done via/at ingress?
    -- This was previously enabled only for oauth-proxy, but there is similar
    -- code in nginx.conf that was executed only for api-proxy but now executes
    -- for both.
    -- TODO: Rationalize OPTIONS method processing: do this only for OPTIONS requests (not all),
    -- and do it all in one place (here or nginx.conf) for both api-proxy and oauth-proxy traffic
    ngx.header["Access-Control-Allow-Headers"] = "authorization"

    if ngx.req.get_method() == "OPTIONS" then
        ngx.status = 200
        ngx.exit(ngx.HTTP_OK)
    end

    -- determine backend and set backend parameters

    local backend = auth.getBackendNameFromIngressHost(ingressHost)
    local backendUrl = auth.getBackendServerUrlFromName(backend)

    auth.info("Processing request for backend '"..ingressHost.."'")

    local authHeader = ngx.req.get_headers()["authorization"]
    local token = nil
    if authHeader then
        if auth.hasCredentialType(authHeader, 'Bearer') then
            token = auth.handleBearerToken(authHeader)
        elseif auth.hasCredentialType(authHeader, 'Basic') then
            token = auth.handleBasicAuth(authHeader)
        end
        if not token then
            auth.info("No recognized credentials in authorization header")
        end
    else
        auth.info("No authorization header found")
        if auth.requestUriMatches(ngx.var.request_uri, callbackPath) then
            -- we initiated authentication via pkce, and OP is delivering the code
            -- will redirect to target url, where token will be found in cookie
            auth.oidcHandleCallback()
        elseif auth.requestUriMatches(ngx.var.request_uri, logoutCallbackPath) then
            -- logout was triggered, and OP (always?) is calling our logout URL
            auth.oidcHandleLogoutCallback()
        end
        -- no token yet, and the request is not progressing an OIDC flow.
        -- check if caller has an existing session with a valid token.
        token = auth.getTokenFromSession()

        -- still no token? redirect to OP to authenticate user
        if not token then
            -- no token, redirect to OP to authenticate
            auth.oidcAuthenticate()
        end
    end

    if not token then
        auth.unauthorized("Not authenticated")
    end

    -- token will be an id token except when console calls api proxy, then it's an access token
    if not auth.isAuthorized(token) then
        auth.forbidden("Not authorized")
    end

    if backend == 'verrazzano' then
        local args = ngx.req.get_uri_args()
        if args.cluster then
            -- returns remote cluster server URL
            backendUrl = auth.handleExternalAPICall(token)
        else
            auth.handleLocalAPICall(token)
        end
    else
        if auth.hasCredentialType(authHeader, 'Bearer') then
            -- clear the auth header if it's a bearer token
            ngx.req.clear_header("Authorization")
        end
        -- set the oidc_user
        ngx.var.oidc_user = auth.usernameFromIdToken(token)
        auth.info("Authorized: oidc_user is "..ngx.var.oidc_user)
    end

    auth.info("Setting backend_server_url to '"..backendUrl.."'")
    ngx.var.backend_server_url = backendUrl
`
