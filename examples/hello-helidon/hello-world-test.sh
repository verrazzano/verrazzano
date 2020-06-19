#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

set -xeu

kubectl -n verrazzano-system wait --for=condition=ready pods -l app=verrazzano-operator --timeout 2m
kubectl -n verrazzano-system wait --for=condition=ready pods -l name=verrazzano-validation --timeout 2m

kubectl apply -f ${SCRIPT_DIR}/hello-world-model.yaml
kubectl apply -f ${SCRIPT_DIR}/hello-world-binding.yaml

timeout 10m bash -c 'until kubectl get pods -n greet | grep NAME; do sleep 10; done'
kubectl wait --for=condition=ready pods -n greet --all --timeout 5m

CLUSTER_TYPE=${CLUSTER_TYPE:=OKE}
if [ ${CLUSTER_TYPE} == "OKE" ]; then
  SERVER=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  PORT=80
elif [ ${CLUSTER_TYPE} == "KIND" ]; then
  SERVER=$(kubectl get node ${CLUSTER_NAME}-control-plane -o json | jq -r '.status.addresses[] | select (.type == "InternalIP") | .address')
  PORT=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq '.spec.ports[] | select(.port == 80) | .nodePort')
fi

curl --connect-timeout 30 --retry 10 --retry-delay 30 -X GET http://"${SERVER}":"${PORT}"/greet
