#!/bin/bash
#
# Copyright (c) 2021, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

ADDRESS_RANGE=${1:-"172.18.0.230-172.18.0.254"}
CLUSTER_NAME=${CLUSTER_NAME:-verrazzano}

# Kind load the MetalLB images
# We should ignore these errors because they are not blocking for most pipelines
kind load docker-image quay.io/metallb/controller:v0.13.7 --name "${CLUSTER_NAME}" || true
kind load docker-image quay.io/metallb/speaker:v0.13.7 --name "${CLUSTER_NAME}" || true

# Apply the MetalLB manifest
if [ -f "$HOME"/metallb-native.yaml ] ; then
  kubectl apply -f "$HOME"/metallb-native.yaml --wait=true
else
  wget https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml
  sed -i -e "s|log-level=info|log-level=debug|g" metallb-native.yaml
  sed -i -e "s|failureThreshold: 3|failureThreshold: 6|g" metallb-native.yaml
  kubectl apply -f metallb-native.yaml --wait=true
fi

sleep 5 # wait a few before checking the status, sometimes we get some resource errors on MacOS if we check too soon
kubectl rollout status -n metallb-system deployment controller --timeout=600s
kubectl rollout status -n metallb-system daemonset speaker --timeout=600s

kubectl set resources daemonset -n metallb-system speaker --limits memory=256Mi,cpu=200m

# Create the IPAddressPool for the cluster
kubectl apply -f - <<-EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: vzlocalpool
  namespace: metallb-system
spec:
  addresses:
  - ${ADDRESS_RANGE}
EOF

# Create the L2Advertisment resource for the cluster
kubectl apply -f - <<-EOF
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: vzmetallb
  namespace: metallb-system
spec:
  ipAddressPools:
  - vzlocalpool
EOF

