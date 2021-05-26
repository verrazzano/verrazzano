// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

// OidcAuthLuaFilename defines the auth.lua file name in OIDC proxy ConfigMap
const OidcAuthLuaFilename = "auth.lua"

// OidcAuthLuaFileTemplate is the content of auth.lua file in OIDC proxy ConfigMap
const OidcAuthLuaFileTemplate = `|
    local me = {}
    local random = require("resty.random")
    local base64 = require("ngx.base64")
    local cjson = require "cjson"
    local jwt = require "resty.jwt"

    function me.config(opts)
        for key, val in pairs(opts) do
            me[key] = val
        end
        local aes = require "resty.aes"
        me.aes256 = aes:new(me.cookieKey, nil, aes.cipher(256))
        return me
    end

    function me.log(logLevel, msg, name, value)
        local logObj = {message = msg}
        if name then
            logObj[name] = value
        end
        ngx.log(logLevel,  cjson.encode(logObj))
    end

    function me.logJson(logLevel, msg, err)
        if err then
            me.log(logLevel, msg, 'error', err)
        else
            me.log(logLevel, msg)
        end
    end

    function me.info(msg, obj)
        if obj then
            me.log(ngx.INFO, msg, 'object', obj)
        else
            me.log(ngx.INFO, msg)
        end
    end

    function me.queryParams(req_uri)
         local i = req_uri:find("?")
         if not i then
             i = 0
         else
             i = i + 1
         end
         return ngx.decode_args(req_uri:sub(i), 0)
    end

    function me.query(req_uri, name)
        local i = req_uri:find("&"..name.."=")
        if not i then
        i = req_uri:find("?"..name.."=")
        end
        if not i then
            return nil
        else
            local begin = i+2+name:len()
            local endin = req_uri:find("&", begin)
            if not endin then
                return req_uri:sub(begin)
            end
            return req_uri:sub(begin, endin-1)
        end
    end

    function me.unauthorized(msg, err)
        me.deleteCookie("authn")
        ngx.status = ngx.HTTP_UNAUTHORIZED
        me.logJson(ngx.ERR, msg, err)
        ngx.exit(ngx.HTTP_UNAUTHORIZED)
    end

    function me.forbidden(msg, err)
        ngx.status = ngx.HTTP_FORBIDDEN
        me.logJson(ngx.ERR, msg, err)
        ngx.exit(ngx.HTTP_FORBIDDEN)
    end

    function me.logout()
        local redirectArgs = ngx.encode_args({
            redirect_uri = me.logoutCallbackUri
        })
        local redirURL = me.oidcProviderUri.."/protocol/openid-connect/logout?"..redirectArgs
        ngx.redirect(redirURL)
    end

    function me.randomBase64(size)
        local randBytes = random.bytes(size)
        return base64.encode_base64url(randBytes)
    end

    function me.read_file(path)
        local file = io.open(path, "rb")
        if not file then return nil end
        local content = file:read "*a"
        file:close()
        return content
    end

    -- this should only happen when the console calls the api-proxy
    -- console sends the access token by itself (originally obtained via pkce client)
    function me.handleBearerToken(authHeader)
        me.info("Checking for bearer token")
        local found, index = authHeader:find('Bearer')
        if found then
            local token = string.sub(authHeader, index+2)
            if token
                me.info("Found bearer token in authorization header")
                local err = me.oidcValidateToken(token)
                if err not nil then
                    me.unauthorized("Invalid token")
                end
                return token
            else
                me.unauthorized("Missing token in authorization header)
            end
        end
        me.info "No bearer token found")
        return nil
    end

    local basicCache = {}

    -- should only be called if some vz process is trying to access vmi using basic auth
    -- we use local cache keyed by the auth credentials to avoid calling OP every time
    -- TODO: this seems bad, it means we keep every caller's creds unencrypted in memory until they're pushed out of the cache)
    function me.handleBasicAuth(authHeader)
        me.info("Checking for basic auth credentials")
        local found, index = authHeader:find('Basic')
        if not found then
            me.info("No basic auth credentials found")
            return nil
        end
        local basicCred = string.sub(authHeader, index+2)
        if not basicCred
            me.unauthorized("Invalid BasicAuth authorization header")
        end
        me.info("Found basic auth credentials in authorization header")
        local now = ngx.time()
        local basicAuth = basicCache[basicCred]
        if basicAuth and (now < basicAuth.expiry) then
            me.info("Returning cached token")
            return basicAuth.id_token
        end
        local decode = ngx.decode_base64(basicCred)
        local found = decode:find(':')
        if not found then
            me.unauthorized("Invalid BasicAuth authorization header")
        end
        local u = decode:sub(1, found-1)
        local p = decode:sub(found+1)
        local tokenRes = me.oidcGetTokenWithBasicAuth(u, p)
        local expires_in = tonumber(tokenRes.expires_in)
        for key, val in pairs(basicCache) do
            if val.expiry and now > val.expiry then
                basicCache[key] = nil
            end
        end
        -- TODO: need to adjust for the amount of time that has passed since token issue + clock skew (or just take false cache hits)
        basicCache[basicCred] = {
            -- access_token = tokenRes.access_token,
            id_token = tokenRes.id_token,
            expiry = now + expires_in
        }
        return tokenRes.id_token
    end

    function me.oidcAuthenticate()
        local sha256 = (require 'resty.sha256'):new()
        local codeVerifier = me.randomBase64(32)
        sha256:update(codeVerifier)
        local codeChallenge = base64.encode_base64url(sha256:final())
        local state = me.randomBase64(32)
        local nonce = me.randomBase64(32)
        local stateData = {
            state = state,
            request_uri = ngx.var.request_uri,
            code_verifier = codeVerifier,
            code_challenge = codeChallenge,
            nonce = nonce
        }
        local redirectArgs = ngx.encode_args({
            client_id = me.oidcClient,
            response_type = 'code',
            scope = 'openid',
            code_challenge_method = 'S256',
            code_challenge = codeChallenge,
            state = state,
            nonce = nonce,
            redirect_uri = me.callbackUri
        })
        local redirtURL = me.oidcProviderAuthUri..'?'..redirectArgs
        me.setCookie("state", stateData, me.authStateTtlInSec)
        ngx.header["Cache-Control"] = "no-cache, no-store, max-age=0"
        ngx.redirect(redirtURL)
    end

    -- TODO: clean up cookies
    function me.oidcHandleCallback()
        local queryParams = me.queryParams(ngx.var.request_uri)
        local state = queryParams.state
        local code = queryParams.code
        local nonce = queryParams.nonce
        local cookie = me.readCookie("state")
        if not cookie then
            me.log(ngx.ERR, "Missing callback state redirect to authenticate")
            me.oidcAuthenticate()
        end
        me.deleteCookie("state")
        local stateCk = cookie.state
        -- local nonceCk = cookie.nonce
        local request_uri = cookie.request_uri

        if (state == nil) or (stateCk == nil) then
            me.log(ngx.ERR, "Missing callback state redirect to authenticate")
            me.oidcAuthenticate()
        else
            if state ~= stateCk then
                me.log(ngx.ERR, "Invalid callback state redirect to authenticate")
                me.oidcAuthenticate()
            end
            if not cookie.code_verifier then
                me.log(ngx.ERR, "Invalid callback state redirect to authenticate")
                me.oidcAuthenticate()
            end
            local tokenRes = me.oidcGetTokenWithCode(code, cookie.code_verifier, me.callbackUri)
            if tokenRes then
                me.tokenToCookie(tokenRes)
                ngx.redirect(request_uri)
            end
            me.unauthorized("Failed to obtain token with code)
        end
    end

    function me.oidcHandleLogoutCallback()
        auth.deleteCookie()
        auth.unauthorized("User logged out")
    end

    function me.oidcTokenRequest(formArgs)
        local tokenUri = me.oidcProviderUri.."/protocol/openid-connect/token"
        if me.oidcProviderInClusterUri and me.oidcProviderInClusterUri ~= "" then
            tokenUri = me.oidcProviderInClusterUri.."/protocol/openid-connect/token"
        end
        local http = require "resty.http"
        local httpc = http.new()
        local res, err = httpc:request_uri(tokenUri, {
            method = "POST",
            body = ngx.encode_args(formArgs),
            headers = {
                ["Content-Type"] = "application/x-www-form-urlencoded",
            }
        })
        -- ,keepalive_timeout = 60000,
        -- keepalive_pool = 10
        local tokenRes = cjson.decode(res.body)
        if tokenRes.error or tokenRes.error_description then
            me.unauthorized(tokenRes.error_description)
        end
        return tokenRes
    end

    function me.oidcGetTokenWithBasicAuth(u, p)
        return me.oidcTokenRequest({
                        grant_type = 'password',
                        scope = 'openid',
                        client_id = me.oidcDirectAccessClient,
                        password = p,
                        username = u
                    })
    end

    function me.oidcGetTokenWithCode(code, verifier, callbackUri)
        return me.oidcTokenRequest({
                        grant_type = 'authorization_code',
                        client_id = me.oidcClient,
                        code = code,
                        code_verifier = cookie.code_verifier,
                        redirect_uri = me.callbackUri
                    })
    end

    function me.oidcRefreshToken(rft, callbackUri)
        return me.oidcTokenRequest({
                        grant_type = 'refresh_token',
                        client_id = me.oidcClient,
                        refresh_token = rft,
                        redirect_uri = callbackUri
                    })
    end

    -- returns nil if successful, or err otherwise
    function me.oidcValidateToken(token) {
        if not (token) then
            me.unauthorized("Invalid bearer token in authorization header")
        end
        me.logJson(ngx.INFO, "Validate JWT token.")
        local jwt = require "resty.jwt"
        local jwt_obj = jwt:load_jwt(token)
        if (not jwt_obj.header) or (not jwt_obj.header.kid) then
            me.unauthorized("Invalid JWT token", jwt_obj.reason)
        end
        local publicKey = me.publicKey(jwt_obj.header.kid)
        if not publicKey then
            me.unauthorized("No public_key retrieved from keycloak")
        end
        local verified = jwt:verify_jwt_obj(publicKey, jwt_obj)
        if (tostring(jwt_obj.valid) == "false" or tostring(jwt_obj.verified) == "false") then
            me.unauthorized("Invalid JWT token", jwt_obj.reason)
        end
    end

    function me.isAuthorized(idToken)
        me.info("Checking for required role 'requiredRole'")
        local id_token = jwt:load_jwt(idToken)
        if id_token and id_token.payload and id_token.payload.realm_access and id_token.payload.realm_access.roles then
            for _, role in ipairs(id_token.payload.realm_access.roles) do
                if role == requiredRole then
                    return true
                end
            end
        end
        return false
    end

    function me.usernameFromIdToken(idToken)
        local id_token = jwt:load_jwt(idToken)
        if id_token and id_token.payload and id_token.payload.preferred_username then
            return id_token.payload.preferred_username
        end
        return ""
    end

{{- if eq .Mode "api-proxy" }}
    function impersonateKubernetesUser(token) {
        me.logJson(ngx.INFO, "Read service account token.")
        local serviceAccountToken = read_file("/run/secrets/kubernetes.io/serviceaccount/token");
        if not (serviceAccountToken) then
          ngx.status = 401
          me.logJson(ngx.ERR, "No service account token present in pod.")
          ngx.exit(ngx.HTTP_UNAUTHORIZED)
        end
        me.logJson(ngx.INFO, "Set headers")
        ngx.req.set_header("Authorization", "Bearer " .. serviceAccountToken)
        if ( jwt_obj.payload and jwt_obj.payload.groups) then
          me.logJson(ngx.INFO, ("Adding groups " .. cjson.encode(jwt_obj.payload.groups)))
          ngx.req.set_header("Impersonate-Group", jwt_obj.payload.groups)
        end
        if ( jwt_obj.payload and jwt_obj.payload.sub) then
          me.logJson(ngx.INFO, ("Adding sub " .. jwt_obj.payload.sub))
          ngx.req.set_header("Impersonate-User", jwt_obj.payload.sub)
        end
    end
{{- end }}

    -- returns id token, token is refreshed first, if necessary.
    -- nil token returned if no session or the refresh token has expired
    function me.getTokenFromSession()
        local ck = me.readCookie("authn")
        if ck then
            local rft = ck.rt
            local now = ngx.time()
            local expiry = tonumber(ck.expiry)
            local refresh_expiry = tonumber(ck.refresh_expiry)
            if now < expiry then
                return ck.it
            else if now < refresh_expiry then
                local tokenRes = me.oidcRefreshToken(rft, me.callbackUri)
                if tokenRes then
                    local err = me.oidcValidateToken(tokenRes.idt)
                    if err then
                        me.unauthorized("Invalid token")
                    end
                    me.tokenToCookie(tokenRes)
                    me.info("token refreshed",  tokenRes)
                    return tokenRes.idt
                end
            end
            -- no valid token found, delete cookie
            me.deleteCookie("authn")
        end
        return nil
    end

    function me.tokenToCookie(tokenRes)
        -- Do we need access_token? too big > 4k
        local cookiePairs = {
            rt = tokenRes.refresh_token,
            -- at = tokenRes.access_token,
            it = tokenRes.id_token
        }
        local id_token = jwt:load_jwt(tokenRes.id_token)
        local expires_in = tonumber(tokenRes.expires_in)
        local refresh_expires_in = tonumber(tokenRes.refresh_expires_in)
        local now = ngx.time()
        local issued_at = now
        if id_token and id_token.payload then
            if id_token.payload.iat then
                issued_at = tonumber(id_token.payload.iat)
            else
                if id_token.payload.auth_time then
                    issued_at = tonumber(id_token.payload.auth_time)
                end
            end
            --if id_token.payload.email then
            --    cookiePairs.email = id_token.payload.email
            --end
        end
        local skew = now - issued_at
        -- Expire 30 secs before actual time
        local expiryBuffer = 30
        cookiePairs.expiry = now + expires_in - skew - expiryBuffer
        cookiePairs.refresh_expiry = now + refresh_expires_in - skew - expiryBuffer
        me.setCookie("authn", cookiePairs, tonumber(tokenRes.refresh_expires_in)-expiryBuffer)
    end

    function me.setCookie(ckName, cookiePairs, expiresInSec)
        local expires = ngx.cookie_time(ngx.time() + expiresInSec)
        local attributes = "; Path=/; Secure; HttpOnly; Expires="..expires
        local encrypted = me.aes256:encrypt(cjson.encode(cookiePairs))
        local cookie = base64.encode_base64url(encrypted)
        ngx.header["Set-Cookie"] = ckName..'='..cookie..attributes
    end

    function me.deleteCookie(ckName)
        ngx.header["Set-Cookie"] = ckName..'=; Path=/; Secure; HttpOnly; Expires=Thu, 01 Jan 1970 00:00:00 UTC;'
    end

    function me.readCookie(ckName)
        if not ckName then
            return nil
        end
        local cookie, err = require("resty.cookie"):new()
        local ck = cookie:get(ckName)
        if not ck then
            return nil
        end
        local decoded = base64.decode_base64url(ck)
        if not decoded then
            return nil
        end
        local json = me.aes256:decrypt(decoded)
        if not json then
            return nil
        end
        return cjson.decode(json)
    end

    -- TODO: shouldn't cache these forever
    local certs = {}
    function me.realmCerts(kid)
        local pk = certs[kid]
        if pk then
            return pk
        end
        local http = require "resty.http"
        local httpc = http.new()
        local providerUri = me.oidcProviderUri
        if me.oidcProviderInClusterUri and me.oidcProviderInClusterUri ~= "" then
            providerUri = me.oidcProviderInClusterUri
        end
        local certsUri = providerUri..'/protocol/openid-connect/certs'
        local res, err = httpc:request_uri(certsUri)
        if err then
            return nil
        end
        local data = cjson.decode(res.body)
        if not (data.keys) then
            return nil
        end
        for i, key in pairs(data.keys) do
            if key.kid and key.x5c then
                certs[key.kid] = key.x5c
            end
        end
        return certs[kid]
    end

    function me.publicKey(kid)
        local x5c = me.realmCerts(kid)
        if (not x5c) or (#x5c == 0) then
            return nil
        end
        return "-----BEGIN CERTIFICATE-----\n"..x5c[1].."\n-----END CERTIFICATE-----"
    end

    return me
`
