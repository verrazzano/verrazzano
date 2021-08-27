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
const OidcConfLuaFileTemplate = `local ingressUri = 'https://'..ngx.var.server_name
    # can we trust server_name? it's been legit matched by a server config in nginx.conf
    # but we're wildcarding the backend_name so we can use one server config for all the
    # vmi backends. probably safe if we also check for a recognized backend_name, as we
    # do later on. That check should be done before any token processing, though.
    local callbackPath = "{{ .OidcCallbackPath }}"
    local logoutCallbackPath = "{{ .OidcLogoutCallbackPath }}"

    local auth = require("auth").config({
        callbackUri = ingressUri..callbackPath
        logoutCallbackUri = ingressUri..logoutCallbackPath
    })

    auth.info("Processing request ...")

    # TODO: is this needed here? Is it better done via/at ingress?
    # was previously only for oauth-proxy
    ngx.header["Access-Control-Allow-Headers"] = "authorization"

    if ngx.req.get_method() == "OPTIONS" then
        ngx.status = 200
        ngx.exit(ngx.HTTP_OK)
    end

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

    if ngx.var.backend_name == 'verrazzano-api' then
        local args = ngx.req.get_uri_args()
        if args.cluster then
            auth.handleExternalAPICall(token)
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

        -- set the backend_name and backend_port
        ngx.var.backend_name = 'vmi-system-' .. ngx.var.backend_name
        if ngx.var.backend_name == 'verrazzano-console' then
            ngx.var.backend_port = '8000'
        elseif ngx.var.backend_name == 'grafana' then
            ngx.var.backend_port = '3000'
        elseif ngx.var.backend_name == 'prometheus' then
            ngx.var.backend_port = '9090'
        elseif ngx.var.backend_name == 'kibana' then
            ngx.var.backend_port = '5601'
        elseif ngx.var.backend_name == 'elasticsearch' then
            ngx.var.backend_name = 'vmi-system-' .. 'es-ingest'
            ngx.var.backend_port = '9200'
        else
            # TODO: consider checking for this condition up front, before we've done all the authn/authz processing
            # Would a 500 error be more appropriate?
            me.not_found("Invalid backend name '"..ngx.var.backend_name.."'")
        end
    end
`
