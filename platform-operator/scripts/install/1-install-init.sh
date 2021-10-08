#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

set -eu

# Label the kube-system namespace so that we can apply network policies
log "Adding label needed by network policies to kube-system namespace"
kubectl label namespace kube-system "verrazzano.io/namespace=kube-system" --overwrite
if [ $? -ne 0 ]; then
  echo "Failed to label kube-system namespace"
  exit 1
fi
