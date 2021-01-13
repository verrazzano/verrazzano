#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

NAMESPACE="bobs-books"

echo "Removing Bob's Books OAM example."

kubectl delete -f ${SCRIPT_DIR}/
helm uninstall coherence-operator --namespace ${NAMESPACE}
kubectl delete secret mysql-credentials -n ${NAMESPACE}
kubectl delete secret bobs-bookstore-runtime-encrypt-secret -n ${NAMESPACE}
kubectl delete secret bobs-bookstore-weblogic-credentials -n ${NAMESPACE}
kubectl delete secret bobbys-front-end-runtime-encrypt-secret -n ${NAMESPACE}
kubectl delete secret bobbys-front-end-weblogic-credentials -n ${NAMESPACE}
kubectl delete secret github-packages -n ${NAMESPACE}
kubectl delete secret ocr -n ${NAMESPACE}

echo "Wait for termination of application pods."
attempt=1
while true; do
  count=$(kubectl get pods -n "${NAMESPACE}" 2> /dev/null | wc -l)
  if [ $count -eq 0 ]; then
    echo "No application pods found on attempt ${attempt}."
    break
  elif [ ${attempt} -eq 1 ]; then
    echo "Application pods found on initial attempt. Retrying after delay."
  elif [ ${attempt} -ge 60 ]; then
    echo "ERROR: Application pods found after ${attempt} attempts. Listing pods and exiting."
    kubectl get pods -n "${NAMESPACE}"
    exit 1
  fi
  attempt=$(($attempt+1))
  sleep 1
done

kubectl delete ns ${NAMESPACE}

echo "Removal of Bob's Books OAM example is complete."
