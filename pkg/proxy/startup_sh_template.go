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

// OidcStartupFilename defines the startup.sh file name in OIDC proxy ConfigMap
const OidcStartupFilename = "startup.sh"

// OidcStartupFileTemplate is the template of startup.sh file in OIDC proxy ConfigMap
const OidcStartupFileTemplate = `#!/bin/bash
    startupDir=$(dirname $0)
    cd $startupDir
    cp $startupDir/nginx.conf /etc/nginx/nginx.conf
    cp $startupDir/auth.lua /etc/nginx/auth.lua
    cp $startupDir/conf.lua /etc/nginx/conf.lua
    nameserver=$(grep -i nameserver /etc/resolv.conf | awk '{split($0,line," "); print line[2]}')
    sed -i -e "s|_NAMESERVER_|${nameserver}|g" /etc/nginx/nginx.conf

    mkdir -p /etc/nginx/logs
    touch /etc/nginx/logs/error.log

    export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

{{- if eq .Mode "api-proxy" }}
    cat /etc/ssl/certs/ca-bundle.crt > /etc/nginx/upstream.pem
{{- else if eq .Mode "oauth-proxy" }}
    # Create a pem file that contains all well known ca certs plus
    # the ca-bundle from the managed cluster registration secret.
    # It is valid for ca-bundle to be empty.
    cat /etc/ssl/certs/ca-bundle.crt > /etc/nginx/all-ca-certs.pem
    cat /secret/ca-bundle >> /etc/nginx/all-ca-certs.pem
{{- end }}

    /usr/local/nginx/sbin/nginx -c /etc/nginx/nginx.conf -p /etc/nginx -t
    /usr/local/nginx/sbin/nginx -c /etc/nginx/nginx.conf -p /etc/nginx

{{- if eq .Mode "oauth-proxy" }}
    while [ $? -ne 0 ]; do
        sleep 20
        echo "retry nginx startup ..."
        /usr/local/nginx/sbin/nginx -c /etc/nginx/nginx.conf -p /etc/nginx
    done
{{- else if eq .Mode "api-proxy" }}
    sh -c "$startupDir/reload.sh &"
{{- end }}
    tail -f /etc/nginx/logs/error.log
`
