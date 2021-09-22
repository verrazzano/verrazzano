#!/bin/bash

#
# Copyright (c) 2021, Oracle and/or its affiliates.
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

kubectl create -f https://download.elastic.co/downloads/eck/1.7.0/crds.yaml
kubectl apply -f https://download.elastic.co/downloads/eck/1.7.0/operator.yaml

cat <<EOF | kubectl apply -f -
apiVersion: elasticsearch.k8s.elastic.co/v1
kind: Elasticsearch
metadata:
  name: quickstart
spec:
  version: 7.14.0
  nodeSets:
  - name: default
    count: 1
    config:
      node.store.allow_mmap: false
  http:
    service:
      spec:
        type: LoadBalancer
    tls:
      selfSignedCertificate:
        subjectAltNames:
        - ip: 172.18.0.230
        - ip: 172.18.0.231
        - ip: 172.18.0.232
EOF

retries=0
until kubectl get secret quickstart-es-http-certs-public ; do
  retries=$(($retries+1))
  sleep 10
  if [ "$retries" -ge 30 ] ; then
    break
  fi
done
if [ "$retries" -ge 30 ] ; then
  log "can not find secret quickstart-es-http-certs-public"
  return 1
fi

kubectl get secret quickstart-es-http-certs-public -o go-template='{{index .data "ca.crt" | base64decode }}' > ${SCRIPT_DIR}/ca-bundle

retries=0
until kubectl get secret quickstart-es-elastic-user ; do
  retries=$(($retries+1))
  sleep 10
  if [ "$retries" -ge 30 ] ; then
    break
  fi
done
if [ "$retries" -ge 30 ] ; then
  log "can not find secret quickstart-es-elastic-user"
  return 1
fi

kubectl create ns verrazzano-install

kubectl -n verrazzano-install create secret generic external-es-secret --from-literal=username=elastic --from-literal=password=$(kubectl get secret quickstart-es-elastic-user -o go-template='{{.data.elastic | base64decode}}') --from-file=${SCRIPT_DIR}/ca-bundle
