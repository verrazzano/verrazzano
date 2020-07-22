#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

set -euo pipefail

echo "Installing Helidon hello world application."

echo "Wait for Verrazzano operator to be ready."
kubectl -n verrazzano-system wait --for=condition=ready pods -l app=verrazzano-operator --timeout 2m
echo "Wait for Verrazzano validation to be ready."
kubectl -n verrazzano-system wait --for=condition=ready pods -l name=verrazzano-validation --timeout 2m

echo "Apply application model."
kubectl apply -f ${SCRIPT_DIR}/hello-world-model.yaml
echo "Apply application binding."
kubectl apply -f ${SCRIPT_DIR}/hello-world-binding.yaml

echo "Wait for application namespace to be active."
attempt=1
while true; do
  status=$(kubectl get ns -o=jsonpath='{.items[?(@.metadata.name=="greet")].status.phase}' || true)
  if [ "${status}" == "Active" ]; then
    echo "Application namespace found and active on attempt ${attempt}, namespace status \"${status}\"."
    break
  elif [ ${attempt} -ge 60 ]; then
    echo "ERROR: Application namespace not found on final attempt ${attempt}, namespace status \"${status}\". Exiting."
    exit 1
  elif [ ${attempt} -eq 1 ]; then
    echo "Application namespace not found on initial attempt, namespace status \"${status}\". Retrying after delay."
  fi
  attempt=$(($attempt+1))
  sleep .5
done

echo "Wait for application pods to be running."
attempt=1
while true; do
  # Can't use kubectl wait with timeout as this fails immediately if there are no pods.
  # xargs is used to trim whitespace from value
  count=$( (kubectl get pods -n greet -o=jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' || true) | wc -w | xargs)
  if [ ${count} -ge 2 ]; then
    echo "Application pods found and running on attempt ${attempt}, pod count ${count}."
    break
  elif [ ${attempt} -ge 60 ]; then
    echo "ERROR: Application pods not found on final attempt ${attempt}, pod count ${count}. Exiting."
    exit 1
  elif [ ${attempt} -eq 1 ]; then
    echo "Application pods not found on initial attempt, pod count ${count}. Retrying after delay."
  fi
  attempt=$((attempt+1))
  sleep 5
done

echo "Determine application endpoint."
CLUSTER_TYPE=${CLUSTER_TYPE:=OKE}
if [ "${CLUSTER_TYPE}" == "OKE" ]; then
  SERVER=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  PORT=80
elif [ "${CLUSTER_TYPE}" == "KIND" ]; then
  SERVER=$(kubectl get node ${KIND_CLUSTER_NAME}-control-plane -o json | jq -r '.status.addresses[] | select (.type == "InternalIP") | .address')
  PORT=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq '.spec.ports[] | select(.port == 80) | .nodePort')
elif [ "${CLUSTER_TYPE}" == "OLCNE" ]; then
  SERVER=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  if [ "${SERVER}" == "null" ]; then
      SERVER=$(kubectl get service -n istio-system istio-ingressgateway -o json | jq -r '.spec.externalIPs[0]')
  fi
  PORT=80
fi

url="http://${SERVER}:${PORT}/greet"
expect="Hello World"
echo "Connect to application endpoint ${url}"
reply=$(curl -s --connect-timeout 30 --retry 10 --retry-delay 30 -X GET ${url})
code=$?
if [ ${code} -ne 0 ]; then
  echo "ERROR: Application connection failed: ${code}. Exiting."
  exit ${code}
elif [[ "$reply" != *"${expect}"* ]]; then
  echo "ERROR: Application reply unexpected: ${reply}, expected: ${expect}. Exiting."
  exit 1
else
  echo "Application reply correct: ${reply}"
fi

echo "Installation of Helidon hello world application successful."
