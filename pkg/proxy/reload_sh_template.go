// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

//
// reload.sh file is used only by api-proxy
//

// OidcReloadFilename defines the reload.sh file name in OIDC proxy ConfigMap
const OidcReloadFilename = "reload.sh"

// OidcReloadFileTemplate is the template of reload.sh file in OIDC proxy ConfigMap
const OidcReloadFileTemplate = `|
    #!/bin/bash

    function reload() {
      nginx -t -p /etc/nginx
      if [ $? -eq 0 ]
      then
        echo "Detected Nginx Configuration Change"
        echo "Executing: nginx -s reload -p /etc/nginx"
        nginx -s reload -p /etc/nginx
      fi
    }

    while true
    do
    if [[ -L /etc/nginx/ca-bundle && -f /api-config/admin-ca-bundle && -s /api-config/admin-ca-bundle && \
          "$(realpath /etc/nginx/ca-bundle)" != "/api-config/admin-ca-bundle" ]]; then
      rm -rf /etc/nginx/ca-bundle
      ln -s /api-config/admin-ca-bundle /etc/nginx/ca-bundle
      reload
    fi
    if [[ -L /etc/nginx/ca-bundle && ! -f /api-config/admin-ca-bundle && \
          "$(realpath /etc/nginx/ca-bundle)" == "/api-config/admin-ca-bundle" && \
          -f /api-config/default-ca-bundle && -s /api-config/default-ca-bundle ]]; then
      rm -rf /etc/nginx/ca-bundle
      ln -s /api-config/default-ca-bundle /etc/nginx/ca-bundle
      reload
    fi
    if [[ ! -L /etc/nginx/ca-bundle && -f /api-config/admin-ca-bundle && -s /api-config/admin-ca-bundle ]]; then
      ln -s /api-config/admin-ca-bundle /etc/nginx/ca-bundle
      reload
    fi
    if [[ ! -L /etc/nginx/ca-bundle && -f /api-config/default-ca-bundle && -s /api-config/default-ca-bundle ]]; then
      ln -s /api-config/default-ca-bundle /etc/nginx/ca-bundle
      reload
    fi
    sleep 5
    done
`
