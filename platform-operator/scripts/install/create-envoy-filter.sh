#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. ${SCRIPT_DIR}/logging.sh

CONFIG_DIR=$SCRIPT_DIR/config
TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

set -ueo pipefail

function create_envoy_filter {

    log "Adding Istio server header network filter"
    kubectl apply -f <(echo "
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: server-header-filter
  namespace: istio-system
spec:
  configPatches:
    - applyTo: NETWORK_FILTER
      match:
        listener:
          filterChain:
            filter:
              name: envoy.filters.network.http_connection_manager
      patch:
        operation: MERGE
        value:
          typed_config:
            '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
            server_header_transformation: PASS_THROUGH
")
}

# Create certificates and istio secret to hold certificates if we haven't already
if ! kubectl get envoyfilter server-header-filter -n istio-system > /dev/null 2>&1 ; then
  echo "Creating envoy filter"
  create_envoy_filter
  if [ $? -ne 0 ]; then
    echo "Failed to create envoy filter"
    exit 1
  fi
fi
