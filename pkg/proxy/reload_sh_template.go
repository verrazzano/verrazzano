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

    adminCABundleMD5=""
    defaultCABundleMD5=""
    upstreamCACertFile="/etc/nginx/upstream.pem"
    localClusterCACertFile="/api-config/default-ca-bundle"
    adminClusterCACertFile="/api-config/admin-ca-bundle"
    defaultCACertFile="/etc/ssl/certs/ca-bundle.crt"
    tmpUpstreamCACertFile="/tmp/upstream.pem"
    maxSizeTrustedCertsFileDefault=$(echo $((10*1024*1024)))
    if [[ ! -z "${MAX_SIZE_TRUSTED_CERTS_FILE}" ]]; then
        maxSizeTrustedCertsFileDefault=${MAX_SIZE_TRUSTED_CERTS_FILE}
    fi

    function reload() {
        nginx -t -p /etc/nginx
        if [ $? -eq 0 ]
        then
            echo "Detected Nginx Configuration Change"
            echo "Executing: nginx -s reload -p /etc/nginx"
            nginx -s reload -p /etc/nginx
        fi
    }

    function reset_md5() {
        adminCABundleMD5=""
        defaultCABundleMD5=""
    }

    function local_cert_config() {
        if [[ -s $localClusterCACertFile ]]; then
            md5Hash=$(md5sum "$localClusterCACertFile")
            if [ "$defaultCABundleMD5" != "$md5Hash" ] ; then
                echo "Adding local CA cert to $upstreamCACertFile"
                cat $upstreamCACertFile > $tmpUpstreamCACertFile
                cat $localClusterCACertFile > $upstreamCACertFile
                cat $tmpUpstreamCACertFile >> $upstreamCACertFile
                rm -rf $tmpUpstreamCACertFile
                defaultCABundleMD5="$md5Hash"
                reload
            fi
        fi
    }

    function admin_cluster_cert_config() {
        if [[ -s $adminClusterCACertFile ]]; then
            md5Hash=$(md5sum "$adminClusterCACertFile")
            if [ "$adminCABundleMD5" != "$md5Hash" ] ; then
                echo "Adding admin cluster CA cert to $upstreamCACertFile"
                cat $upstreamCACertFile > $tmpUpstreamCACertFile
                cat $adminClusterCACertFile > $upstreamCACertFile
                cat $tmpUpstreamCACertFile >> $upstreamCACertFile
                rm -rf $tmpUpstreamCACertFile
                adminCABundleMD5="$md5Hash"
                reload
            fi
        else 
            if [ "$adminCABundleMD5" != "" ] ; then
                reset_md5
                local_cert_config
            fi 
        fi
    }

    function default_cert_config() {
        cat $defaultCACertFile > $upstreamCACertFile
    }

    while true
    do
        trustedCertsFileSize=$(wc -c < $upstreamCACertFile)
        if [ $trustedCertsFileSize -ge $maxSizeTrustedCertsFileDefault ] ; then
            echo "$upstreamCACertFile file size greater than  $maxSizeTrustedCertsFileDefault, resetting.."
            reset_md5
            default_cert_config
        fi

        local_cert_config
        admin_cluster_cert_config
        sleep .1
    done
`
