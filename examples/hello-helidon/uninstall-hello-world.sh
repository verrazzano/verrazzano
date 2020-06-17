#!/usr/bin/env bash

SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

kubectl delete -f ${SCRIPT_DIR}/hello-world-binding.yaml
kubectl delete -f ${SCRIPT_DIR}/hello-world-model.yaml
