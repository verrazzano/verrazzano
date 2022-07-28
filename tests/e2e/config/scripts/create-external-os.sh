#!/bin/bash

#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ "$EXTERNAL_ELASTICSEARCH" != "true" ]; then
  echo "Skipping creating external Elasticsearch when not using EXTERNAL_ELASTICSEARCH"
  exit 0
fi

if [ "$CLUSTER_NUMBER" != "1" ]; then
  echo "Skipping creating external Elasticsearch on a managed cluster"
  exit 0
fi

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

# This corresponds to OpenSearch 1.2.3
OPENSEARCH_CHART_VERSION="1.5.7"

# Install OpenSearch
helm repo add opensearch https://opensearch-project.github.io/helm-charts/
helm repo update
helm upgrade --install opensearch opensearch/opensearch --version "$OPENSEARCH_CHART_VERSION" \
  -f "$SCRIPT_DIR"/opensearch.yaml

# Discover the LoadBalancer IP
until [ -n "$(kubectl get svc opensearch-cluster-master -o jsonpath='{.status.loadBalancer.ingress[0].ip}')" ]; do
    sleep 3
done

EXTERNAL_IP="$(kubectl get svc opensearch-cluster-master -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"

echo "bootstrapping certificates for LoadBalancer @ $EXTERNAL_IP"

sed -i "s/subjectAltName = critical,IP:.*/subjectAltName = critical,IP:$EXTERNAL_IP/" "$SCRIPT_DIR"/cert.conf
# create root ca key
echo -n '' > index.txt
echo -n '00' > serial.txt
openssl genrsa -out root-key.pem 2048
openssl req -x509 -new -config "$SCRIPT_DIR"/root.conf -key root-key.pem -out root-ca.pem -batch
openssl genrsa -out server_key.pem 2048
openssl req -new -config "$SCRIPT_DIR"/cert.conf -key server_key.pem -out cert.csr -batch
openssl ca -config "$SCRIPT_DIR"/root.conf -keyfile root-key.pem -cert root-ca.pem \
  -policy signing_policy -extensions signing_node_req \
  -in cert.csr -out cert.pem -outdir "$SCRIPT_DIR" -batch -keyform PEM
openssl pkcs8 -topk8 -inform PEM -in server_key.pem -out key.pem -nocrypt
certdata=$(cat cert.pem)
echo "-----${certdata#*-----}" > cert.pem

# this secret is used by OpenSearch for loading certificates
kubectl create secret generic opensearch-certificates \
  --from-file=cert.pem \
  --from-file=key.pem \
  --from-file=root-ca.pem

helm upgrade --install opensearch opensearch/opensearch --version "$OPENSEARCH_CHART_VERSION" \
  -f "$SCRIPT_DIR"/opensearch.yaml \
  --set service.loadBalancerIP="$EXTERNAL_IP"

kubectl get namespace -o=name | grep "verrazzano-install"
if [ $? -ne 0 ]; then
  echo "External OpenSearch - Create the verrazzano-install namespace"
  kubectl create namespace verrazzano-install
fi
cp root-ca.pem "$SCRIPT_DIR"/ca-bundle
cat cert.pem >> "$SCRIPT_DIR"/ca-bundle

# this secret is used by Verrazzano for loading certificates and credentials
kubectl -n verrazzano-install create secret generic external-es-secret --from-literal=username=admin --from-literal=password=admin --from-file="${SCRIPT_DIR}"/ca-bundle
