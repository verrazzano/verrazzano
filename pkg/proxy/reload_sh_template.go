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

//
// reload.sh file is used only by api-proxy
//

// OidcReloadFilename defines the reload.sh file name in OIDC proxy ConfigMap
const OidcReloadFilename = "reload.sh"

// OidcReloadFileTemplate is the template of reload.sh file in OIDC proxy ConfigMap
const OidcReloadFileTemplate = `    #!/bin/bash

    function reload() {
      nginx -t -p /etc/nginx
      if [ $? -eq 0 ]
      then
        echo "Detected Nginx Configuration Change"
        echo "Executing: nginx -s reload -p /etc/nginx"
        nginx -s reload -p /etc/nginx
      fi
    }

    adminCABundleMD5=""
    defaultCABundleMD5=""
    maxSizeTrustedCertsFileDefault=$(echo $((10*1024*1024)))
    if [[ ! -z "${MAX_SIZE_TRUSTED_CERTS_FILE}" ]]; then
        maxSizeTrustedCertsFileDefault=${MAX_SIZE_TRUSTED_CERTS_FILE}
    fi

    while true
    do
        trustedCertsFileSize=$(wc -c < /etc/nginx/upstream.pem)
        if [ $trustedCertsFileSize -ge $maxSizeTrustedCertsFileDefault ] ; then
            cat /etc/ssl/certs/ca-bundle.crt > /etc/nginx/upstream.pem
            adminCABundleMD5=""
            defaultCABundleMD5=""
        fi

        if [[ -s /api-config/admin-ca-bundle ]]; then
            md5Hash=$(md5sum "/api-config/admin-ca-bundle")
            if [ "$adminCABundleMD5" != "$md5Hash" ] ; then
                cat /api-config/admin-ca-bundle >> /etc/nginx/upstream.pem
                adminCABundleMD5="$md5Hash"
                reload
            fi
        fi

        if [[ -s /api-config/default-ca-bundle ]]; then
            md5Hash=$(md5sum "/api-config/default-ca-bundle")
            if [ "$defaultCABundleMD5" != "$md5Hash" ] ; then
                cat /api-config/default-ca-bundle >> /etc/nginx/upstream.pem
                defaultCABundleMD5="$md5Hash"
                reload
            fi
        fi

        sleep 5
    done
`
