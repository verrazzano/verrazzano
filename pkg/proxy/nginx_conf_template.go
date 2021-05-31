// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

// OidcNginxConfFilename defines the nginx.lua file name in OIDC proxy ConfigMap
const OidcNginxConfFilename = "nginx.conf"

// OidcNginxConfFileTemplate is the template of nginx.conf file in OIDC proxy ConfigMap
const OidcNginxConfFileTemplate = `|
    #user  nobody;
    worker_processes  1;

    #error_log  logs/error.log;
    #error_log  logs/error.log  notice;
    #error_log  logs/error.log  info;

    pid        logs/nginx.pid;
    {{- if eq .Mode "api-proxy" }}
    env KUBERNETES_SERVICE_HOST;
    env KUBERNETES_SERVICE_PORT;
    {{- end }}

    events {
        worker_connections  1024;
    }

    http {
        include       mime.types;
        default_type  application/octet-stream;

        #log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
        #                  '$status $body_bytes_sent "$http_referer" '
        #                  '"$http_user_agent" "$http_x_forwarded_for"';

        error_log  logs/error.log  info;

        sendfile        on;
        #tcp_nopush     on;

{{- if eq .Mode "oauth-proxy" }}
        client_max_body_size 65m;
{{- end }}

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
        lua_ssl_verify_depth 2;
{{- if eq .Mode "api-proxy" }}
        lua_ssl_trusted_certificate /etc/nginx/ca-bundle;
{{- else if eq .Mode "oauth-proxy" }}
        lua_ssl_trusted_certificate /secret/ca-bundle;

        upstream http_backend {
            server {{ .Host }}:{{ .Port }} fail_timeout=30s max_fails=10;
        }
{{- end }}

        server {
            listen       8775;
            server_name  apiserver;
            root     /opt/nginx/html;
            #charset koi8-r;

{{- if eq .Mode "oauth-proxy" }}
            set $oidc_user "";
{{- end }}
            rewrite_by_lua_file /etc/nginx/conf.lua;
{{- if eq .Mode "api-proxy" }}
            set_by_lua $kubernetes_server_host 'return os.getenv("KUBERNETES_SERVICE_HOST")';
            set_by_lua $kubernetes_server_port 'return os.getenv("KUBERNETES_SERVICE_PORT")';
{{- end }}

            #access_log  logs/host.access.log  main;
            expires           0;
            add_header        Cache-Control private;

{{- if eq .Mode "oauth-proxy" }}
            proxy_set_header  X-WEBAUTH-USER $oidc_user;
{{- end }}

            location / {
{{- if eq .Mode "oauth-proxy" }}
                proxy_pass http://http_backend;
{{- else if eq .Mode "api-proxy" }}
                proxy_pass https://$kubernetes_server_host:$kubernetes_server_port;
{{- end }}
            }

            error_page 404 /404.html;
                location = /40x.html {
            }

            #error_page  404              /404.html;
            # redirect server error pages to the static page /50x.html
            #
            error_page   500 502 503 504  /50x.html;
            location = /50x.html {
                root   html;
            }
        }
    }
`
