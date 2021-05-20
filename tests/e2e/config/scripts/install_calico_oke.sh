#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

CALICO_VERSION=$(grep ‘calico-version=’ ${SCRIPT_DIR}/../../../../.third-party-test-versions | sed ‘s/calico-version=//g’)

echo "Install Calico ${CALICO_VERSION}..."

echo "FORCING FAILURE TEMPORARY"
CALICO_VERSION=""

# CALICO_HOME must be set to the location of the downloaded Calico bundle
CALICO_YAML=${CALICO_HOME}/${CALICO_VERSION}/k8s-manifests/calico-policy-only.yaml

# uncomment the pod CIDR setting and replace the default pod CIDR with the OKE pod CIDR default
sed -i -e "s?# - name: CALICO_IPV4POOL_CIDR?- name: CALICO_IPV4POOL_CIDR?" ${CALICO_YAML}
sed -i -e "s?#   value: \"192.168.0.0/16\"?  value: \"10.244.0.0/16\"?" ${CALICO_YAML}
grep -B 1 -A 2 POOL_CIDR ${CALICO_YAML}

echo "Applying Calico YAML: ${CALICO_YAML}"
kubectl apply -f ${CALICO_YAML}
if [ $? -ne 0 ]; then
    exit 1
fi
