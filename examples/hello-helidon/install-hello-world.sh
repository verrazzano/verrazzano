#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

set -eu

kubectl -n verrazzano-system wait --for=condition=ready pods -l app=verrazzano-operator --timeout 2m
kubectl -n verrazzano-system wait --for=condition=ready pods -l name=verrazzano-validation --timeout 2m

kubectl apply -f ${SCRIPT_DIR}/hello-world-model.yaml
kubectl apply -f ${SCRIPT_DIR}/hello-world-binding.yaml

retries=0
until [ "$retries" -ge 60 ]
do
    if kubectl get namespace greet > /dev/null 2>&1 ; then
        break
    fi
    sleep .5
done

retries=0
until [ "$retries" -ge 60 ]
do
   kubectl get pods -n greet | grep NAME && break
   retries=$(($retries+1))
   sleep 5
done

kubectl wait --for=condition=ready pods -n greet --all --timeout 5m

CLUSTER_TYPE=${CLUSTER_TYPE:=OKE}
if [ ${CLUSTER_TYPE} == "OKE" ] || [ "${CLUSTER_TYPE}" == "OLCNE" ]; then
  SERVER=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  PORT=80
elif [ ${CLUSTER_TYPE} == "KIND" ]; then
  SERVER=$(kubectl get node ${KIND_CLUSTER_NAME}-control-plane -o json | jq -r '.status.addresses[] | select (.type == "InternalIP") | .address')
  PORT=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq '.spec.ports[] | select(.port == 80) | .nodePort')
fi

curl --connect-timeout 30 --retry 10 --retry-delay 30 -X GET http://"${SERVER}":"${PORT}"/greet
