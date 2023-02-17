#!/bin/bash
#
# Copyright (c) 2021, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

ADDRESS_RANGE=${1:-"172.18.0.230-172.18.0.254"}

# Apply the MetalLB manifest
wget https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml
sed -i "s|log-level=info|log-level=debug|g" metallb-native.yaml
sed -i "s|failureThreshold: 3|failureThreshold: 6|g" metallb-native.yaml
kubectl apply -f metallb-native.yaml --wait=true
# Wait for the controller. webhook, and speaker to become ready
kubectl wait --namespace metallb-system \
                --for=condition=ready pod \
                --selector=component=controller \
                --timeout=600s
kubectl wait --namespace metallb-system \
                --for=condition=ready pod \
                --selector=component=speaker \
                --timeout=600s

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

