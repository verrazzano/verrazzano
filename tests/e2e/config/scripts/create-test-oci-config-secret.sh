#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

set -e

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true' EXIT

if [ "${OCI_DNS_AUTH}" != "instance_principal" ]; then
  # perform these validations when instance principal is not used
  # Validate expected environment variables exist
  if [ -z "${OCI_CLI_REGION}" ]; then
    echo "OCI_CLI_REGION environment variable must be set"
    exit 1
  fi
  if [ -z "${OCI_CLI_TENANCY}" ]; then
    echo "OCI_CLI_TENANCY environment variable must be set"
    exit 1
  fi
  if [ -z "${OCI_CLI_USER}" ]; then
    echo "OCI_CLI_USER environment variable must be set"
    exit 1
  fi
  if [ -z "${OCI_CLI_FINGERPRINT}" ]; then
    echo "OCI_CLI_FINGERPRINT environment variable must be set"
    exit 1
  fi
  if [ -z "${OCI_CLI_KEY_FILE}" ]; then
    echo "OCI_CLI_KEY_FILE environment variable must be set"
    exit 1
  fi
fi

if [ -z "$WORKSPACE" ] ; then
  echo "This script must only be called from Jenkins and requires environment variable WORKSPACE is set"
  exit 1
fi

# Copy/download the create_oci_config_secret.sh to WORKSPACE and run it as a standalone script
cp ${WORKSPACE}/platform-operator/scripts/install/create_oci_config_secret.sh ${WORKSPACE}/create_oci_config_secret.sh
chmod +x ${WORKSPACE}/create_oci_config_secret.sh

if [ "${OCI_DNS_AUTH}" != "instance_principal" ]; then
  OUTPUT_FILE="$TMP_DIR/oci_config"
  KEY_FILE="$TMP_DIR/oci_key"
  echo "[DEFAULT]" > $OUTPUT_FILE
  echo "#region=someregion.region.com" >> $OUTPUT_FILE
  echo "#tenancy=OCID of the tenancy" >> $OUTPUT_FILE
  echo "#user=" >> $OUTPUT_FILE
  echo "region=${OCI_CLI_REGION}" >> $OUTPUT_FILE
  echo "tenancy=${OCI_CLI_TENANCY}" >> $OUTPUT_FILE
  echo "user=${OCI_CLI_USER}" >> $OUTPUT_FILE
  echo "fingerprint=${OCI_CLI_FINGERPRINT}" >> $OUTPUT_FILE
  if [[ ! -z "${OCI_PRIVATE_KEY_PASSPHRASE}" ]]; then
    echo "  passphrase: ${OCI_PRIVATE_KEY_PASSPHRASE}" >> $OUTPUT_FILE
  fi
  echo "key_file=$KEY_FILE" >> $OUTPUT_FILE
  cat ${OCI_CLI_KEY_FILE} > $KEY_FILE
  echo "Creating the secret with auth_type user_principal"
  ${WORKSPACE}/create_oci_config_secret.sh -o ${OUTPUT_FILE}
else
  echo "Creating the secret with auth_type instance_principal"
  ${WORKSPACE}/create_oci_config_secret.sh -a instance_principal
fi
