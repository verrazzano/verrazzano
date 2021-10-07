#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. ${SCRIPT_DIR}/logging.sh

CONFIG_DIR=$SCRIPT_DIR/config
TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true; _logging_exit_handler $rc' EXIT

set -ueo pipefail

function create_istio_cert_secret {
  CERTS_OUT=$SCRIPT_DIR/build/istio-certs

  rm -rf $CERTS_OUT || true
  rm -f ./index.txt* serial serial.old || true

  mkdir -p $CERTS_OUT
  touch ./index.txt
  echo 1000 > ./serial

  log "Generating CA bundle for Istio"

  # Create the private key for the root CA
  openssl genrsa -out $CERTS_OUT/root-key.pem 4096 || return $?

  # Generate a root CA with the private key
  openssl req -config $CONFIG_DIR/istio_root_ca_config.txt -key $CERTS_OUT/root-key.pem -new -x509 -days 7300 -sha256 -extensions v3_ca -out $CERTS_OUT/root-cert.pem || return $?

  # Create the private key for the intermediate CA
  openssl genrsa -out $CERTS_OUT/ca-key.pem 4096 || return $?

  # Generate certificate signing request (CSR)
  openssl req -config $CONFIG_DIR/istio_intermediate_ca_config.txt -new -sha256 -key $CERTS_OUT/ca-key.pem -out $CERTS_OUT/intermediate-csr.pem || return $?

  # create intermediate cert using the root CA
  openssl ca -batch -config $CONFIG_DIR/istio_root_ca_config.txt -extensions v3_intermediate_ca -days 3650 -notext -md sha256 \
      -keyfile $CERTS_OUT/root-key.pem \
      -cert $CERTS_OUT/root-cert.pem \
      -in $CERTS_OUT/intermediate-csr.pem \
      -out $CERTS_OUT/ca-cert.pem \
      -outdir $CERTS_OUT || return $?

  # Create certificate chain file
  cat $CERTS_OUT/ca-cert.pem $CERTS_OUT/root-cert.pem > $CERTS_OUT/cert-chain.pem || return $?

  kubectl create secret generic cacerts -n istio-system \
      --from-file=$CERTS_OUT/ca-cert.pem \
      --from-file=$CERTS_OUT/ca-key.pem  \
      --from-file=$CERTS_OUT/root-cert.pem \
      --from-file=$CERTS_OUT/cert-chain.pem || return $?

  rm -rf $CERTS_OUT
  rm -f ./index.txt* serial serial.old

  return 0
}

# Create certificates and istio secret to hold certificates if we haven't already
if ! kubectl get secret cacerts -n istio-system > /dev/null 2>&1 ; then
  echo "Creating Istio secret"
  create_istio_cert_secret
  if [ $? -ne 0 ]; then
    echo "Failed to create Istio certificate"
    exit 1
  fi
fi
