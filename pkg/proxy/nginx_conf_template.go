// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

// OidcSslVerifyOptions defines the ssl verify option
const OidcSslVerifyOptions = "lua_ssl_verify_depth 2;"

// OidcSslTrustedOptions defines the ssl trusted certificates option
const OidcSslTrustedOptions = "lua_ssl_trusted_certificate /secret/ca-bundle;"

// OidcNginxConf defines the nginx.lua file name in OIDC proxy ConfigMap
const OidcNginxConfFilename = "nginx.conf"

// OidcNginxConfTemp is the template of nginx.conf file in OIDC proxy ConfigMap
const OidcNginxConfFileTemplate = `|
#user  nobody;
worker_processes  1;

#error_log  logs/error.log;
#error_log  logs/error.log  notice;
#error_log  logs/error.log  info;

pid        logs/nginx.pid;

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

    # TODO: api proxy doesn't have client_max_body_size - should it?
    client_max_body_size 65m;
    #keepalive_timeout  0;
    keepalive_timeout  65;

    #gzip  on;
    #
    lua_package_path '/usr/local/share/lua/5.1/?.lua;;';
    lua_package_cpath '/usr/local/lib/lua/5.1/?.so;;';
    resolver _NAMESERVER_;
    # cache for discovery metadata documents
    lua_shared_dict discovery 1m;
    #  cache for JWKs
    lua_shared_dict jwks 1m;
    # TODO: ssl params hard-wired for api proxy
    {{ .SSLVerifyOptions }}
    {{ .SSLTrustedCAOptions }}

    # TODO: upstream http_backend not present in api-proxy
    upstream http_backend {
        server {{ .Host }}:{{ .Port }} fail_timeout=30s max_fails=10;
    }
    server {
        listen       8775;
        server_name  apiserver;
        root     /opt/nginx/html;
        #charset koi8-r;

        set $oidc_user "";
        rewrite_by_lua_file /etc/nginx/conf.lua;
        #access_log  logs/host.access.log  main;
        expires           0;
        add_header        Cache-Control private;

        # TODO: this is not present in api-proxy
        proxy_set_header  X-WEBAUTH-USER $oidc_user;

        location / {
            proxy_pass http://http_backend;
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
