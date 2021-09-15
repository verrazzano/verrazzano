#!/bin/bash

#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

EXTERNAL_ES_USERNAME=es-username
EXTERNAL_ES_PASSWORD=es-password

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

kubectl create ns verrazzano-system
kubectl create ns verrazzano-install
kubectl create ns cert-manager

# create auth secret for ingress, external-es-auth-secret
# requires htpassword
htpasswd -bc ${SCRIPT_DIR}/auth ${EXTERNAL_ES_USERNAME} ${EXTERNAL_ES_PASSWORD}
kubectl -n verrazzano-system create secret generic external-es-auth-secret --from-file=${SCRIPT_DIR}/auth

# create CA for cluster-issuer, external-es-root-ca
openssl genrsa -out ${SCRIPT_DIR}/tls.key 2048
openssl req -x509 -new -nodes -key ${SCRIPT_DIR}/tls.key -subj "/CN=*.nip.io" -days 3650 -reqexts v3_req -extensions v3_ca -out ${SCRIPT_DIR}/tls.crt
kubectl -n cert-manager create secret tls external-es-root-ca --cert=${SCRIPT_DIR}/tls.crt --key=${SCRIPT_DIR}/tls.key

# create ES secret used by Verrazzao CR
kubectl -n verrazzano-install create secret generic ac-es --from-literal=username=${EXTERNAL_ES_USERNAME} --from-literal=password=${EXTERNAL_ES_PASSWORD} --from-file=ca-bundle=${SCRIPT_DIR}/tls.crt