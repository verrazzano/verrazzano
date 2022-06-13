#!/bin/bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

ADDRESS_RANGE=${1:-"172.18.0.230-172.18.0.254"}
WORKSPACE=${WORKSPACE:-"."}

METALLB_MANIFEST=${WORKSPACE}/metallb.yaml

kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.11.0/manifests/namespace.yaml
curl --silent https://raw.githubusercontent.com/metallb/metallb/v0.11.0/manifests/metallb.yaml | sed 's/imagePullPolicy: Always/imagePullPolicy: IfNotPresent/g' > ${METALLB_MANIFEST}
kubectl apply -f ${METALLB_MANIFEST}
kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)" || true
kubectl apply -f - <<-EOF
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: my-ip-space
      protocol: layer2
      addresses:
      - ${ADDRESS_RANGE}
EOF
