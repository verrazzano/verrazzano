#!/bin/bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

set -u
ENV_ERROR=false
if [ -z "${VERRAZZANO_KUBECONFIG:-}" ] ; then
    echo "Environment variable VERRAZZANO_KUBECONFIG must be set an point to a valid kubeconfig"
    ENV_ERROR=true
fi
if [ -z "${DNS_TYPE:-}" ]; then
    echo "Environment variable DNS_TYPE must be set either oci or xip.io"
    ENV_ERROR=true
else 
    if [ $DNS_TYPE != "xip.io" ] && [ $DNS_TYPE != "oci" ]; then
       echo "Environment variable DNS_TYPE $DNS_TYPE must be set either oci or xip.io"
       ENV_ERROR=true
    fi
    if [ $DNS_TYPE = "oci" ]; then
       . $SCRIPT_DIR/check-ocidns-env.sh || exit 1
    fi
fi
if [ $ENV_ERROR = true ]; then
    exit 1
fi

if [ ! -d "${SCRIPT_DIR}/.verrazzano" ] ; then
    mkdir -p ${SCRIPT_DIR}/.verrazzano
fi

export CLUSTER_TYPE=OKE

. $SCRIPT_DIR/common.sh


$SCRIPT_DIR/1-install-istio.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR
if [ $DNS_TYPE = "xip.io" ]; then
  $SCRIPT_DIR/2a-install-system-components-magicdns.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR
else
  $SCRIPT_DIR/2b-install-system-components-ocidns.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR
fi
$SCRIPT_DIR/3-install-verrazzano.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR
$SCRIPT_DIR/4-install-keycloak.sh >&$CONSOLE_STDOUT 2>&$CONSOLE_STDERR

#
# Wait for environment to be ready
kubectl -n verrazzano-system wait --for=condition=ready pods -l app=verrazzano-operator --timeout 2m
kubectl -n verrazzano-system wait --for=condition=ready pods -l name=verrazzano-validation --timeout 2m

consoleout
consoleout "Installation of cluster ${CLUSTER_TYPE} completed"
