#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $0); pwd -P)

kubectl delete -f ${SCRIPT_DIR}/hello-world-binding.yaml --timeout 5m
kubectl delete -f ${SCRIPT_DIR}/hello-world-model.yaml --timeout 2m
