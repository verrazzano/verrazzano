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

// OidcNginxConfFilename defines the nginx.lua file name in OIDC proxy ConfigMap
const OidcNginxConfFilename = "nginx.conf"

// OidcNginxConfFileTemplate is the template of nginx.conf file in OIDC proxy ConfigMap
const OidcNginxConfFileTemplate = `#user  nobody;
    worker_processes  1;

    error_log  logs/error.log info;
    pid        logs/nginx.pid;

    env KUBERNETES_SERVICE_HOST;
    env KUBERNETES_SERVICE_PORT;
    env VZ_API_HOST;
    env VZ_API_VERSION;

    events {
        worker_connections  1024;
    }

    http {
        include       mime.types;
        default_type  application/octet-stream;

        #log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
        #                  '$status $body_bytes_sent "$http_referer" '
        #                  '"$http_user_agent" "$http_x_forwarded_for"';

        sendfile        on;
        #tcp_nopush     on;

        # TODO: This was previously set but only for oauth-proxy
        # Do we need this here at all, or should we be enforcing
        # this sort of constraint in the ingress controller?
        #
        # client_max_body_size 65m;

        #keepalive_timeout  0;
        keepalive_timeout  65;

        #gzip  on;

        lua_package_path '/usr/local/share/lua/5.1/?.lua;;';
        lua_package_cpath '/usr/local/lib/lua/5.1/?.so;;';
        resolver _NAMESERVER_;

        # cache for discovery metadata documents
        lua_shared_dict discovery 1m;
        #  cache for JWKs
        lua_shared_dict jwks 1m;

        #access_log  logs/host.access.log  main;
        server_tokens off

        #charset koi8-r;
        expires           0;
        add_header        Cache-Control private;

        set $vz_env_dns_suffix "{{ .EnvironmentDnsSuffix }}";

        root     /opt/nginx/html;

        # reject requests with no host header
        server {
            listen      80;
            server_name "";
            return      444;
        }

        # pass-thru servers for keycloak and rancher - no proxy authn/authz
        server {
            listen 8775
            server_name  keycloak.$vz_env_dns_suffix;
            location / {
                proxy_pass keycloak-http.keycloak.svc.cluster.local:80;
            }
        }
        server {
            listen 8775
            server_name  rancher.$vz_env_dns_suffix;
            location / {
                proxy_pass rancher.cattle-system.svc.cluster.local:80;
            }
        }

        # verrazzano api and console
        server {
            listen 8775
            server_name  verrazzano.$vz_env_dns_suffix;

            # api
            location /20210501/ {
                lua_ssl_verify_depth 2;
                lua_ssl_trusted_certificate /etc/nginx/upstream.pem;

                set $backend_name "verrazzano-api"
                set $kubernetes_server_url "";
                rewrite_by_lua_file /etc/nginx/conf.lua;
                proxy_pass $kubernetes_server_url;
                proxy_ssl_trusted_certificate /etc/nginx/upstream.pem;
                header_filter_by_lua_block {
                    local h, _ = ngx.req.get_headers()["origin"]
                    if h and h ~= "*" and  h ~= "null" then
                        ngx.header["Access-Control-Allow-Origin"] = h
                    end
                    ngx.header["Access-Control-Allow-Headers"] = "authorization, content-type"
                }
            }

            # console
            location / {
{{- if eq .SSLEnabled true }}
                lua_ssl_verify_depth 2;
                lua_ssl_trusted_certificate /etc/nginx/all-ca-certs.pem;
{{- end }}
                set $backend_name "verrazzano-console"
                set $backend_port "";
                set $oidc_user "";
                rewrite_by_lua_file /etc/nginx/conf.lua;
                proxy_set_header  X-WEBAUTH-USER $oidc_user;
                proxy_pass $backend_name.verrazzano-system.svc.cluster.local:$backend_port
            }
        }

        # vmi services
        server {
            listen 8775
            server_name  ~<backend_name>.vmi.system.$vz_env_dns_suffix;

{{- if eq .SSLEnabled true }}
            lua_ssl_verify_depth 2;
            lua_ssl_trusted_certificate /etc/nginx/all-ca-certs.pem;
{{- end }}

            # proxy_pass backend previously specified as an upstream server with parameters:
            #     server {{ .Host }}:{{ .Port }} fail_timeout=30s max_fails=10;
            # Are those parameters still needed?

            location / {
                set $backend_port "";
                set $oidc_user "";
                rewrite_by_lua_file /etc/nginx/conf.lua;
                proxy_set_header  X-WEBAUTH-USER $oidc_user;
                proxy_pass $backend_name.verrazzano-system.svc.cluster.local:$backend_port
            }
        }
    }
`
