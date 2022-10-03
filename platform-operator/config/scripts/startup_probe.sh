#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# This script is used to validate if the webhook endpoints are ready and available to serve incoming requests.

WEBHOOK_PORT={$1:-9443}
SCHEME="https"
HOST="localhost"
URL="${SCHEME}://${HOST}:${WEBHOOK_PORT}"
CONTENT_TYPE_HEADER="Content-Type: application/json"

curl -kfs -H ${CONTENT_TYPE_HEADER} -o /dev/null "${URL}/validate-install-verrazzano-io-v1alpha1-verrazzano" \
&& curl -kfs -H ${CONTENT_TYPE_HEADER} -o /dev/null "${URL}/validate-install-verrazzano-io-v1beta1-verrazzano" \
&& curl -kfs -H ${CONTENT_TYPE_HEADER} -o /dev/null "${URL}/validate-clusters-verrazzano-io-v1alpha1-verrazzanomanagedcluster" \
&& curl -kfs -XPOST "${URL}/convert" -o /dev/null -H "${CONTENT_TYPE_HEADER}" -d '{"apiVersion":"apiextensions.k8s.io/v1", "kind":"ConversionReview", "request":{}}'

exit $?