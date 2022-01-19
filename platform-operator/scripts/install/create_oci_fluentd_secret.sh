#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
#
# Creates a Kubernetes secret based on an OCI CLI configuration for consumption by the fluentd OCI plugin
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. ${SCRIPT_DIR}/oci_secret_common.sh

function usage {
    echo
    echo "usage: $0 [-o oci_config_file] [-s config_file_section]"
    echo "  -o oci_config_file         The full path to the OCI configuration file (default ~/.oci/config)"
    echo "  -s config_file_section     The properties section within the OCI configuration file.  Default is DEFAULT"
    echo "  -k secret_name             The secret name containing the OCI configuration.  Default is \"oci-fluentd\""
    echo "  -c context_name            The kubectl context to use"
    echo "  -h                         Help"
    echo
    exit 1
}

OUTPUT_FILE=$TMP_DIR/oci.yaml

OCI_CONFIG_FILE=~/.oci/config
SECTION=DEFAULT
OCI_FLUENTD_SECRET_NAME=oci-fluentd
K8SCONTEXT=""
VERRAZZANO_INSTALL_NS=verrazzano-install

while getopts c:o:s:k:h flag
do
    case "${flag}" in
        o) OCI_CONFIG_FILE=${OPTARG};;
        s) SECTION=${OPTARG};;
        k) OCI_FLUENTD_SECRET_NAME=${OPTARG};;
        c) K8SCONTEXT="--context=${OPTARG}";;
        h) usage;;
        *) usage;;
    esac
done

SECTION_PROPS=$(read_config $OCI_CONFIG_FILE $SECTION *)
eval $SECTION_PROPS

CONFIG_TMP=$TMP_DIR/oci_config_tmp
cat <<EOT > $CONFIG_TMP
[DEFAULT]
user=${user}
tenancy=${tenancy}
region=${region}
fingerprint=${fingerprint}
key_file=/root/.oci/key
EOT

if [[ ! -z "$pass_phrase" ]]; then
echo "pass_phrase=${pass_phrase}" >> CONFIG_TMP
fi

# create the secret in verrazzano-install namespace
kubectl ${K8SCONTEXT} get secret $OCI_FLUENTD_SECRET_NAME -n $VERRAZZANO_INSTALL_NS > /dev/null 2>&1
if [ $? -eq 0 ]; then
  # secret exists
  echo "Secret $OCI_FLUENTD_SECRET_NAME already exists in ${VERRAZZANO_INSTALL_NS} namespace."
  exit 1
fi

# Create the secret
kubectl ${K8SCONTEXT} create secret -n $VERRAZZANO_INSTALL_NS  generic $OCI_FLUENTD_SECRET_NAME --from-file=config=${CONFIG_TMP} \
  --from-file=key=${key_file}
