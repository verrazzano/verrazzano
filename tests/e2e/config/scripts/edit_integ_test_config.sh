#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Edit a given input test configuration file

INPUT_CONFIG_FILE=$1

CONSOLE_HOST="$(kubectl get ingress verrazzano-ingress -n verrazzano-system -o jsonpath='{.spec.rules[0].host}')"
CONSOLE_URL="https://${CONSOLE_HOST}"
CONSOLE_PWD="$(kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode)"
cat "${INPUT_CONFIG_FILE}" | jq  --arg url "${CONSOLE_URL}" --arg user "verrazzano" --arg pwd "${CONSOLE_PWD}" '.driverInfo.url = $url | .loginInfo.username = $user | .loginInfo.password = $pwd'
