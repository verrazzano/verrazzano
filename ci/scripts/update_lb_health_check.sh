#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Script to update load balancer health check with the given IPs
# Usage - ./update_lb_health_check.sh "1.1.1.1 2.2.2.2"

IP=$1
for ip in $IP; do
    echo $ip
    LB_OCID=$(oci network public-ip get --public-ip-address $ip | jq -r '.data."display-name"' |  awk '{print $(NF)}')
    echo $LB_OCID
    oci lb health-checker update --load-balancer-id $LB_OCID --backend-set-name TCP-443 \
    --port 0 --protocol TCP --return-code 200 --retries 3 --response-body-regex ".*" \
    --timeout-in-millis 3000 --interval-in-millis 10000 --wait-for-state SUCCEEDED
done
