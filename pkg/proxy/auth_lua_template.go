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

// OidcAuthLuaFilename defines the auth.lua file name in OIDC proxy ConfigMap
const OidcAuthLuaFilename = "auth.lua"

// OidcAuthLuaFileTemplate is the content of auth.lua file in OIDC proxy ConfigMap
const OidcAuthLuaFileTemplate = `local me = {}
    local random = require("resty.random")
    local base64 = require("ngx.base64")
    local cjson = require "cjson"
    local jwt = require "resty.jwt"
    local validators = require "resty.jwt-validators"

    local oidcRealm = "{{ .OidcRealm }}"
    local oidcClient = "{{ .PKCEClientID }}"
    local oidcDirectAccessClient = "{{ .PGClientID }}"
    local requiredRole = "{{ .RequiredRealmRole }}"

    local authStateTtlInSec = tonumber("{{ .AuthnStateTTL }}")

    local oidcProviderHost = "{{ .OidcProviderHost }}"
    local oidcProviderHostInCluster = "{{ .OidcProviderHostInCluster }}"

    local oidcProviderUri = nil
    local oidcProviderInClusterUri = nil
    local oidcIssuerUri = nil
    local oidcIssuerUriLocal = nil

    local logoutCallbackUri = nil

    function me.config(opts)
        for key, val in pairs(opts) do
            me[key] = val
        end
        if not cookiekey then
            -- this is a global variable, initialize it exactly once
            cookiekey = me.randomBase64(32)
        end
        local aes = require "resty.aes"
        me.aes256 = aes:new(cookiekey, nil, aes.cipher(256))
        me.initOidcProviderUris()
        return me
    end

    function me.initOidcProviderUris()
        oidcProviderUri = 'https://'..oidcProviderHost..'/auth/realms/'..oidcRealm
        if oidcProviderHostInCluster and oidcProviderHostInCluster ~= "" then
            oidcProviderInClusterUri = 'http://'..oidcProviderHostInCluster..'/auth/realms/'..oidcRealm
        end
{{ if eq .Mode "oauth-proxy" }}
        logoutCallbackUri = me.ingressUri..me.logoutCallbackPath
{{ else if eq .Mode "api-proxy" }}
        local keycloakURL = me.read_file("/api-config/keycloak-url")
        if keycloakURL and keycloakURL ~= "" then
            me.info("keycloak-url specified in multi-cluster secret, will not use in-cluster oidc provider host.")
            oidcProviderUri = keycloakURL..'/auth/realms/'..oidcRealm
            oidcProviderInClusterUri = nil
        end
{{ end }}
        oidcIssuerUri = oidcProviderUri
        oidcIssuerUriLocal = oidcProviderInClusterUri
        --[[
        if oidcProviderUri then
            me.info("set oidcProviderUri to "..oidcProviderUri)
        end
        if oidcProviderInClusterUri then
            me.info("set oidcProviderInClusterUri to "..oidcProviderInClusterUri)
        end
        if oidcIssuerUri then
            me.info("set oidcIssuerUri to "..oidcIssuerUri)
        end
        if oidcIssuerUri then
            me.info("set oidcIssuerUriLocal to "..oidcIssuerUriLocal)
        end
        --]]
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
        local redirectURL = me.oidcProviderUri.."/protocol/openid-connect/logout?"..redirectArgs
        ngx.redirect(redirectURL)
    end

    function me.randomBase64(size)
        local randBytes = random.bytes(size)
        local encoded = base64.encode_base64url(randBytes)
        return string.sub(encoded, 1, size)
    end

    function me.read_file(path)
        local file = io.open(path, "rb")
        if not file then return nil end
        local content = file:read "*a"
        file:close()
        return content
    end

    function me.write_file(path, data)
      local file = io.open(path, "a+")
      if not file then return nil end
      file:write(data)
      file:close()
    end

    function me.hasCredentialType(authHeader, credentialType)
        local start, end = authHeader:find(credentialType)
        if start > 1 then
            return true
        end
        return false
    end

    -- console sends access token by itself (originally obtained via pkce client)
    function me.handleBearerToken(authHeader)
        local found, index = authHeader:find('Bearer')
        if found then
            local token = string.sub(authHeader, index+2)
            if token then
                me.info("Found bearer token in authorization header")
                me.oidcValidateBearerToken(token)
                return token
            else
                me.unauthorized("Missing token in authorization header")
            end
        end
        return nil
    end

    local basicCache = {}

    -- should only be called if some vz process is trying to access vmi using basic auth
    -- tokens are cached locally
    function me.handleBasicAuth(authHeader)
        -- me.info("Checking for basic auth credentials")
        local found, index = authHeader:find('Basic')
        if not found then
            me.info("No basic auth credentials found")
            return nil
        end
        local basicCred = string.sub(authHeader, index+2)
        if not basicCred then
            me.unauthorized("Invalid BasicAuth authorization header")
        end
        me.info("Found basic auth credentials in authorization header")
        local now = ngx.time()
        local basicAuth = basicCache[basicCred]
        if basicAuth and (now < basicAuth.expiry) then
            me.info("Returning cached token")
            return basicAuth.id_token
        end
        local decode, err = ngx.decode_base64(basicCred)
        if err then
            me.unauthorized("Unable to decode BasicAuth authorization header")
        end
        local found = decode:find(':')
        if not found then
            me.unauthorized("Invalid BasicAuth authorization header")
        end
        local u = decode:sub(1, found-1)
        local p = decode:sub(found+1)
        local tokenRes = me.oidcGetTokenWithBasicAuth(u, p)
        if not tokenRes then
            me.unauthorized("Could not get token")
        end
        me.oidcValidateIDTokenPG(tokenRes.id_token)
        local expires_in = tonumber(tokenRes.expires_in)
        for key, val in pairs(basicCache) do
            if val.expiry and now > val.expiry then
                basicCache[key] = nil
            end
        end
        basicCache[basicCred] = {
            -- access_token = tokenRes.access_token,
            id_token = tokenRes.id_token,
            expiry = now + expires_in
        }
        return tokenRes.id_token
    end

    function me.getOidcProviderUri()
        if oidcProviderUri and oidcProviderUri ~= "" then
            return oidcProviderUri
        else
            return oidcProviderInClusterUri
        end
    end

    function me.getLocalOidcProviderUri()
        if oidcProviderInClusterUri and oidcProviderInClusterUri ~= "" then
            return oidcProviderInClusterUri
        else
            return oidcProviderUri
        end
    end

    function me.getOidcTokenUri()
        return me.getLocalOidcProviderUri().."/protocol/openid-connect/token"
    end

    function me.getOidcCertsUri()
        return me.getLocalOidcProviderUri()..'/protocol/openid-connect/certs'
    end

    function me.getOidcAuthUri()
        return me.getOidcProviderUri()..'/protocol/openid-connect/auth'
    end

    function me.oidcAuthenticate()
        me.info("Authenticating user")
        local sha256 = (require 'resty.sha256'):new()
        -- code verifier must be between 43 and 128 characters
        local codeVerifier = me.randomBase64(56)
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
            client_id = oidcClient,
            response_type = 'code',
            scope = 'openid',
            code_challenge_method = 'S256',
            code_challenge = codeChallenge,
            state = state,
            nonce = nonce,
            redirect_uri = me.callbackUri
        })
        local redirectURL = me.getOidcAuthUri()..'?'..redirectArgs
        me.setCookie("state", stateData, authStateTtlInSec)
        ngx.header["Cache-Control"] = "no-cache, no-store, max-age=0"
        ngx.redirect(redirectURL)
    end

    function me.oidcHandleCallback()
        me.info("Handle authentication callback")
        local queryParams = me.queryParams(ngx.var.request_uri)
        local state = queryParams.state
        local code = queryParams.code
        local nonce = queryParams.nonce
        local cookie = me.readCookie("state")
        if not cookie then
            me.unauthorized("Missing state cookie")
        end
        me.deleteCookie("state")
        local stateCk = cookie.state
        -- local nonceCk = cookie.nonce
        local request_uri = cookie.request_uri

        if (state == nil) or (stateCk == nil) then
            me.unauthorized("Missing callback state")
        else
            if state ~= stateCk then
                me.unauthorized("Invalid callback state")
            end
            if not cookie.code_verifier then
                me.unauthorized("Invalid code_verifier")
            end
            local tokenRes = me.oidcGetTokenWithCode(code, cookie.code_verifier, me.callbackUri)
            if tokenRes then
                me.oidcValidateIDTokenPKCE(tokenRes.id_token)
                me.tokenToCookie(tokenRes)
                ngx.redirect(request_uri)
            end
            me.unauthorized("Failed to obtain token with code")
        end
    end

    function me.oidcHandleLogoutCallback()
        auth.deleteCookie("authn")
        auth.unauthorized("User logged out")
    end

    function me.oidcTokenRequest(formArgs)
        me.info("Requesting token from OP")
        local tokenUri = me.getOidcTokenUri()
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
            me.info("Error requesting token")
            me.unauthorized(tokenRes.error_description)
        end
        return tokenRes
    end

    function me.oidcGetTokenWithBasicAuth(u, p)
        return me.oidcTokenRequest({
                        grant_type = 'password',
                        scope = 'openid',
                        client_id = oidcDirectAccessClient,
                        password = p,
                        username = u
                    })
    end

    function me.oidcGetTokenWithCode(code, verifier, callbackUri)
        return me.oidcTokenRequest({
                        grant_type = 'authorization_code',
                        client_id = oidcClient,
                        code = code,
                        code_verifier = verifier,
                        redirect_uri = callbackUri
                    })
    end

    function me.oidcRefreshToken(rft, callbackUri)
        return me.oidcTokenRequest({
                        grant_type = 'refresh_token',
                        client_id = oidcClient,
                        refresh_token = rft,
                        redirect_uri = callbackUri
                    })
    end

    function me.oidcValidateBearerToken(token)
        -- console sends access tokens obtained via PKCE client
        -- test code sends ID tokens obtained from the PG client
        -- need to accept either type in Authorization header (for now)
        local claim_spec = {
            typ = validators.equals({ "Bearer", "ID" }),
            iss = validators.equals( oidcIssuerUri ),
            azp = validators.equals_any_of({ oidcClient, oidcDirectAccessClient }),
        }
        me.oidcValidateTokenWithClaims(token, claim_spec)
    end

    function me.oidcValidateIDTokenPKCE(token)
        me.oidcValidateToken(token, "ID", oidcIssuerUri, oidcClient)
    end

    function me.oidcValidateIDTokenPG(token)
        if not oidcIssuerUriLocal then
            me.oidcValidateToken(token, "ID", oidcIssuerUri, oidcDirectAccessClient)
        else
            me.oidcValidateToken(token, "ID", oidcIssuerUriLocal, oidcDirectAccessClient)
        end
    end

    function me.oidcValidateToken(token, expectedType, expectedIssuer, clientName)
        if not token or token == "" then
            me.unauthorized("Nil or empty token")
        end
        if not expectedType then
            me.unauthorized("Nil or empty expectedType")
        end
        if not expectedIssuer then
            me.unauthorized("Nil or empty expectedIssuer")
        end
        if not clientName then
            me.unauthorized("Nil or empty clientName")
        end
        local claim_spec = {
            typ = validators.equals( expectedType ),
            iss = validators.equals( expectedIssuer ),
            azp = validators.equals( clientName ),
        }
        me.oidcValidateTokenWithClaims(token, claim_spec)
    end

    function me.oidcValidateTokenWithClaims(token, claim_spec)
        me.info("Validating JWT token")
        local default_claim_spec = {
            iat = validators.is_not_before(),
            exp = validators.is_not_expired(),
            aud = validators.required()
        }
        -- passing verify a function to retrieve key didn't seem to work, so doing load then verify
        local jwt_obj = jwt:load_jwt(token)
        if (not jwt_obj) or (not jwt_obj.header) or (not jwt_obj.header.kid) then
            me.unauthorized("Failed to load token or no kid")
        end
        local publicKey = me.publicKey(jwt_obj.header.kid)
        if not publicKey then
            me.unauthorized("No public key found")
        end
        -- me.info("TOKEN: iss is "..jwt_obj.payload.iss)
        -- me.info("TOKEN: oidcIssuerUri is"..oidcIssuerUri)
        local verified = jwt:verify_jwt_obj(publicKey, jwt_obj, default_claim_spec, claim_spec)
        if not verified or (tostring(jwt_obj.valid) == "false" or tostring(jwt_obj.verified) == "false") then
            me.unauthorized("Failed to validate token", jwt_obj.reason)
        end
    end

    function me.isAuthorized(idToken)
        me.info("Checking for required role '"..requiredRole.."'")
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
        -- me.info("usernameFromIdToken: fetching preferred_username")
        local id_token = jwt:load_jwt(idToken)
        if id_token and id_token.payload and id_token.payload.preferred_username then
            return id_token.payload.preferred_username
        end
        me.unauthorized("usernameFromIdToken: preferred_username not found")
    end

    -- returns id token, token is refreshed first, if necessary.
    -- nil token returned if no session or the refresh token has expired
    function me.getTokenFromSession()
        me.info("Check for existing session")
        local ck = me.readCookie("authn")
        if ck then
            local rft = ck.rt
            local now = ngx.time()
            local expiry = tonumber(ck.expiry)
            local refresh_expiry = tonumber(ck.refresh_expiry)
            if now < expiry then
                return ck.it
            else
                if now < refresh_expiry then
                    local tokenRes = me.oidcRefreshToken(rft, me.callbackUri)
                    if tokenRes then
                        me.oidcValidateIDTokenPKCE(tokenRes.id_token)
                        me.tokenToCookie(tokenRes)
                        me.info("token refreshed",  tokenRes)
                        return tokenRes.idt
                    end
                end
            end
        end
        -- no valid token found, delete cookie
        me.deleteCookie("authn")
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
            -- if id_token.payload.email then
            --     cookiePairs.email = id_token.payload.email
            -- end
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
            me.info("Cookie not found")
            return nil
        end
        local decoded = base64.decode_base64url(ck)
        if not decoded then
            me.info("Cookie not decoded")
            return nil
        end
        local json = me.aes256:decrypt(decoded)
        if not json then
            me.info("Cookie not decrypted")
            return nil
        end
        return cjson.decode(json)
    end

    local certs = {}

    function me.realmCerts(kid)
        local pk = certs[kid]
        if pk then
            return pk
        end
        local http = require "resty.http"
        local httpc = http.new()
        local certsUri = me.getOidcCertsUri()
        local res, err = httpc:request_uri(certsUri)
        if err then
            me.logJson(ngx.WARN, "Could not retrieve certs", err)
            return nil
        end
        local data = cjson.decode(res.body)
        if not (data.keys) then
            me.info("No keys found")
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

{{ if eq .Mode "api-proxy" }}
    local vzApiHost = os.getenv("VZ_API_HOST")
    local vzApiVersion = os.getenv("VZ_API_VERSION")

    function me.getServiceAccountToken()
      me.logJson(ngx.INFO, "Read service account token.")
      local serviceAccountToken = me.read_file("/run/secrets/kubernetes.io/serviceaccount/token")
      if not (serviceAccountToken) then
        ngx.status = 401
        me.logJson(ngx.ERR, "No service account token present in pod.")
        ngx.exit(ngx.HTTP_UNAUTHORIZED)
      end
      return serviceAccountToken
    end

    function me.getLocalServerURL()
      local host = os.getenv("KUBERNETES_SERVICE_HOST")
      local port = os.getenv("KUBERNETES_SERVICE_PORT")
      local serverUrl = "https://" .. host .. ":" .. port
      return serverUrl
    end

    --[[

    -- the next three functions appear to be unused, commenting out

    function me.split(s, delimiter)
      local result = {}
      for match in (s..delimiter):gmatch("(.-)"..delimiter) do
          table.insert(result, match)
      end
      return result
    end

    function me.contains(table, element)
      for _, value in pairs(table) do
        if value == element then
          return true
        end
      end
      return false
    end

    function me.capture(cmd, raw)
      local f = assert(io.popen(cmd, 'r'))
      local s = assert(f:read('*a'))
      f:close()
      if raw then return s end
      s = string.gsub(s, '^%s+', '')
      s = string.gsub(s, '%s+$', '')
      s = string.gsub(s, '[\n\r]+', ' ')
      return s
    end

    --]]

    function me.getK8SResource(resourcePath)
      local http = require "resty.http"
      local httpc = http.new()
      local res, err = httpc:request_uri("https://" .. vzApiHost .. "/" .. vzApiVersion .. resourcePath,{
          headers = {
              ["Authorization"] = ngx.req.get_headers()["authorization"],
          },
      })
      if err then
        ngx.status = 401
        me.logJson(ngx.ERR, "Error accessing vz api", err)
        ngx.exit(ngx.HTTP_UNAUTHORIZED)
      end
      if not(res) or not (res.body) then
        ngx.status = 401
        me.logJson(ngx.ERR, "Unable to get k8s resource.")
        ngx.exit(ngx.HTTP_UNAUTHORIZED)
      end
      local cjson = require "cjson"
      return cjson.decode(res.body)
    end

    function me.getVMC(cluster)
      return me.getK8SResource("/apis/clusters.verrazzano.io/v1alpha1/namespaces/verrazzano-mc/verrazzanomanagedclusters/" .. cluster)
    end

    function me.getSecret(secret)
      return me.getK8SResource("/api/v1/namespaces/verrazzano-mc/secrets/" .. secret)
    end

    function me.handleLocalAPICall(token)
        me.logJson(ngx.INFO, "Read service account token and set auth header.")
        local serviceAccountToken = me.getServiceAccountToken()
        ngx.req.set_header("Authorization", "Bearer " .. serviceAccountToken)
        me.logJson(ngx.INFO, "Set groups and users")
        local jwt_obj = jwt:load_jwt(token)
        if ( jwt_obj.payload and jwt_obj.payload.groups) then
            me.logJson(ngx.INFO, ("Adding groups " .. cjson.encode(jwt_obj.payload.groups)))
            ngx.req.set_header("Impersonate-Group", jwt_obj.payload.groups)
        end
        if ( jwt_obj.payload and jwt_obj.payload.sub) then
            me.logJson(ngx.INFO, ("Adding sub " .. jwt_obj.payload.sub))
            ngx.req.set_header("Impersonate-User", jwt_obj.payload.sub)
        end
        ngx.var.kubernetes_server_url = me.getLocalServerURL()
    end

    function me.handleExternalAPICall(token)
        local args = ngx.req.get_uri_args()
        me.logJson(ngx.INFO, "Read vmc resource for " .. args.cluster)
        local vmc = me.getVMC(args.cluster)
        if not(vmc) or not(vmc.status) or not(vmc.status.apiUrl) then
            ngx.status = 401
            me.logJson(ngx.ERR, "Unable to fetch vmc api url for vmc " .. args.cluster)
            ngx.exit(ngx.HTTP_UNAUTHORIZED)
        end

        -- To access managed cluster api server on self signed certificates, the admin cluster api server needs ca certificates for the managed cluster.
        -- A secret is created in admin cluster during mutli cluster setup that contains the prometheus host and this ca certificate.
        -- Here we read the name of that secret from vmc spec and retrieve the secret from cluster and read the cacrt field.
        -- The value of cacrt field is decoded to get the ca certificate and is appended to file being pointed to by the proxy_ssl_trusted_certificate variable.
        local serverUrl = vmc.status.apiUrl .. "/" .. vzApiVersion
        if not(vmc.spec) or not(vmc.spec.prometheusSecret) then
            ngx.status = 401
            me.logJson(ngx.ERR, "Unable to fetch prometheus secret name for vmc to access cart for managed cluster " .. args.cluster)
            ngx.exit(ngx.HTTP_UNAUTHORIZED)
        end

        local secret = me.getSecret(vmc.spec.prometheusSecret)
        if not(secret) or not(secret.data) or not(secret.data[args.cluster .. ".yaml"]) then
            ngx.status = 401
            me.logJson(ngx.ERR, "Unable to fetch prometheus secret for vmc to access cart for managed cluster " .. args.cluster)
            ngx.exit(ngx.HTTP_UNAUTHORIZED)
        end

        local decodedSecretYaml = ngx.decode_base64(secret.data[args.cluster .. ".yaml"])
        if not(decodedSecretYaml) then
            ngx.status = 401
            me.logJson(ngx.ERR, "Unable to decode prometheus secret for vmc to access cart for managed cluster " .. args.cluster)
            ngx.exit(ngx.HTTP_UNAUTHORIZED)
        end

        local startIndex, endIndex = string.find(decodedSecretYaml, "cacrt: |")
        if startIndex >= 1 and endIndex > startIndex then
            local sub = string.sub(decodedSecretYaml, endIndex+1)
            local startIndex, _ = string.find(sub, "-----BEGIN CERTIFICATE-----")
            local _, endIndex = string.find(sub, "-----END CERTIFICATE-----")
            if startIndex >= 1 and endIndex > startIndex then
                me.write_file("/etc/nginx/upstream.pem", string.sub(sub, startIndex, endIndex))
            end
        end

        args.cluster = nil
        ngx.req.set_uri_args(args)
        ngx.var.kubernetes_server_url = serverUrl .. ngx.var.uri
    end
{{ end }}

    return me
`
