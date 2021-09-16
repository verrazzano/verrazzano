#!/bin/bash

#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ $EXTERNAL_ELASTICSEARCH != "true" ]; then
  echo "Skipping creating external es secret when not using EXTERNAL_ELASTICSEARCH"
  exit 0
fi

if [ $CLUSTER_NUMBER != "1" ]; then
  echo "Skipping creating external es ingress on a managed cluster"
  exit 0
fi

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

kubectl create ns verrazzano-system

# Create cluster-issuer, external-es-cluster-issuer
    kubectl apply -f <(echo "
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: external-es-cluster-issuer
spec:
  ca:
    secretName: external-es-root-ca
")

# create ing, external-es-ingress
kubectl apply -f ${SCRIPT_DIR}/external-es-ingress.yaml
kubectl -n verrazzano-system get ingress external-es-ingress -o yaml
