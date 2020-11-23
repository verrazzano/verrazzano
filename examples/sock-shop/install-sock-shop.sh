#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)
. $SCRIPT_DIR/../../install/logging.sh
. $SCRIPT_DIR/../../install/config.sh

set -euo pipefail

echo "Installing Helidon Sock Shop application."

echo "Wait for Verrazzano operator to be ready."
kubectl -n verrazzano-system wait --for=condition=ready pods -l app=verrazzano-operator --timeout 2m
echo "Wait for Verrazzano validation to be ready."
kubectl -n verrazzano-system wait --for=condition=ready pods -l name=verrazzano-validation --timeout 2m

echo "Apply application model."
kubectl apply -f ${SCRIPT_DIR}/sock-shop-model.yaml
echo "Apply application binding."
kubectl apply -f ${SCRIPT_DIR}/sock-shop-binding.yaml

echo "Wait for application namespace to be active."
attempt=1
while true; do
  status=$(kubectl get ns -o=jsonpath='{.items[?(@.metadata.name=="sockshop")].status.phase}' || true)
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
  count=$( (kubectl get pods -n sockshop -o=jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' || true) | wc -w | xargs)
  if [ ${count} -ge 1 ]; then
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
SERVER=$(get_application_ingress_ip)
PORT=80

url="http://${SERVER}:${PORT}/catalogue"
expect="For all those leg lovers out there."
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
  echo "Sock Shop catalogue reply correct"
fi

echo "Installation of Helidon Sock Shop application successful."
