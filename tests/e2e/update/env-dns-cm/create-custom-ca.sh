#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Script to create a custom CA cert and secret for use with Verrazzano
# - CA is self-signed
secretName=
secretNamespace=
caName=
updateCert=

usage() {
  local ec=${1:-0}
  local msg=${2:-""}
  echo """
usage:

$(basename $0) [-h] [-c ca-name] [-k] [-n secret-namespace] [-s secret-name]

-c Create/update cert
-k Create/update key and CA cert; default if CA cert/key do not exist
-n Secret namespace (default \"customca\")
-s Secret name (default \"[ca-name]-secret\")

-h Print this help text
"""

  if [ ! -z "$msg" ]; then
    echo """
error: $msg
"""
  fi
  exit $ec
}

while getopts 'hc:kn:s:' opt; do
  case $opt in
  c)
    # shellcheck disable=SC2034
    caName=${OPTARG}
    ;;
  k)
    # shellcheck disable=SC2034
    updateCert=true
    ;;
  n)
    secretNamespace=${OPTARG}
    ;;
  s)
    secretName=${OPTARG}
    ;;
  h)
    usage
    ;;
  ?)
    usage 1 "Invalid option: ${OPTARG}"
    ;;
  esac
done


if [ -z "${caName}" ]; then
  usage 1 "Provide a CA name"
fi

if [ -z "${secretName}" ]; then
  secretName=${caName}-secret
fi
if [ -z "${secretNamespace}" ]; then
  secretNamespace="customca"
fi

keyFile=${caName}.key
certFile=${caName}.crt

if [ "${updateCert}" == "true" ] || [ ! -e ${keyfile} ]; then
  echo "Creating key file $keyFile with certificate file $certFile"

  # Generate a CA private key
  openssl genrsa -out ${keyFile} 2048

  # Create a self signed certificate, valid for 10yrs with the 'signing' option set
  openssl req -x509 -new -nodes -key ${keyFile} -subj "/CN=${caName}" -days 3650 -reqexts v3_req -extensions v3_ca -out ${certFile}
fi

echo "Creating secret ${secretNamespace}/${secretName} for CA ${caName}"
if ! kubectl get ns ${secretNamespace} 2>&1 > /dev/null; then
  echo "creating namespace ${secretNamespace}"
  kubectl create ns ${secretNamespace} || true
fi

kubectl create secret tls -n ${secretNamespace} ${secretName} -o yaml --dry-run=client --save-config \
	--cert=${certFile} --key=${keyFile} | kubectl apply -f -

echo "Done"
