
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

TEST_CONFIG_FILE=$1
INGRESS_IP=$(kubectl get svc ingress-controller-ingress-nginx-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
sed -i "s/XX_DNS_ZONE_XX/${INGRESS_IP}.xip.io/" ${TEST_CONFIG_FILE}
