#!/usr/bin/env bash

SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

kubectl delete -f ${SCRIPT_DIR}/hello-world-binding.yaml --timeout 5m
kubectl delete -f ${SCRIPT_DIR}/hello-world-model.yaml --timeout 2m
