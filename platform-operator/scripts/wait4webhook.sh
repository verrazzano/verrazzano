#!/bin/bash
# Copyright (c) 2020, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

function poll-webhook {
    SECONDS=0
    MAX_SECONDS=120
    while [ $SECONDS -lt $MAX_SECONDS ]; do
        status_code=$(curl  -vvv --insecure --silent --output /tmp/out --write-out '%{http_code}' -H 'Content-Type: application/json' $1)
        echo "$1 returned HTTP $status_code."
        if [[ "$status_code" != "200" ]]; then
            cat /tmp/out
            echo "waiting 5 seconds"
            let SECONDS=SECONDS+5
            sleep 5
        else
            exit 0
        fi
    done
    echo "timeout waiting for VPO webhook"
    exit 1
}

poll-webhook "https://verrazzano-platform-operator-webhook.verrazzano-install.svc.cluster.local:443/validate-install-verrazzano-io-v1beta1-verrazzano"
