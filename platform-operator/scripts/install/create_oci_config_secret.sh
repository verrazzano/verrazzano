#!/usr/bin/env bash
#
# Copyright (c) 2020, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Creates a Kubernetes secret based on an OCI CLI configuration for consumption by External-DNS and/or Cert-Manager
#

# WARNING: This script can be downloaded and run standalone. All required functions must exist within this script
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

if [ -z "${KUBECONFIG:-}" ] ; then
  echo "Environment variable KUBECONFIG must be set to a valid kube config file"
  exit 1
fi

TMP_DIR=$(mktemp -d)
trap 'rc=$?; rm -rf ${TMP_DIR} || true' EXIT

# read a config item from a specified section of an oci config file
function read_config() {
  if [[ $# -lt 2 || ! -f $1 ]]; then
    echo "usage: iniget <file> [--list|<SECTION> [key]]"
    return 1
  fi
  local ocifile=$1

  if [ "$2" == "--list" ]; then
    for SECTION in $(cat $ocifile | grep "\[" | sed -e "s#\[##g" | sed -e "s#\]##g"); do
      echo $SECTION
    done
    return 0
  fi

  local SECTION=$2
  local key
  [ $# -eq 3 ] && key=$3

 # Read the lines from the OCI CLI configuration file, by ignoring the comments and prefix each line with the given section.
 local lines=$(awk '!/^#/{gsub(/^[[:space:]]*#.*/,"",$0);print}' $ocifile | awk '/\[/{prefix=$0; next} $1{print prefix $0}')
  for line in $lines; do
    if [[ "$line" = \[$SECTION\]* ]]; then
      local keyval=$(echo $line | sed -e "s/^\[$SECTION\]//")
      if [[ -z "$key" ]]; then
        echo $keyval
      else
        if [[ "$keyval" = $key=* ]]; then
          echo $(echo $keyval | sed -e "s/^$key=//")
        fi
      fi
    fi
  done
}

function usage {
    echo
    echo "usage: $0 [-o oci_config_file] [-s config_file_section]"
    echo "  -o oci_config_file         The full path to the OCI configuration file. Default is ~/.oci/config"
    echo "  -s config_file_section     The properties section within the OCI configuration file. Default is DEFAULT"
    echo "  -k secret_name             The secret name containing the OCI configuration. Default is oci"
    echo "  -c context_name            The kubectl context to use"
    echo "  -a auth_type               The auth_type to be used to access OCI. Valid values are user_principal/instance_principal. Default is user_principal."
    echo "  -n namespace               The target namespace to create the secret in. Default is \"verrazzano-install\"."
    echo "  -h                         Help"
    echo
    exit 1
}

OUTPUT_FILE=$TMP_DIR/oci.yaml

OCI_CONFIG_FILE=~/.oci/config
SECTION=DEFAULT
OCI_CONFIG_SECRET_NAME=oci
K8SCONTEXT=""
TARGET_NS=verrazzano-install
OCI_AUTH_TYPE="user_principal"

while getopts c:n:o:s:k:a:h flag
do
    case "${flag}" in
        o) OCI_CONFIG_FILE=${OPTARG};;
        s) SECTION=${OPTARG};;
        k) OCI_CONFIG_SECRET_NAME=${OPTARG};;
        c) K8SCONTEXT="--context=${OPTARG}";;
        a) OCI_AUTH_TYPE_INPUT=${OPTARG};;
        n) TARGET_NS=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

if [ "${OCI_AUTH_TYPE_INPUT:-}" ] ; then
  if [ ${OCI_AUTH_TYPE_INPUT} == "user_principal" ] || [ ${OCI_AUTH_TYPE_INPUT} == "instance_principal" ]; then
    OCI_AUTH_TYPE=${OCI_AUTH_TYPE_INPUT}
  fi
fi

if [ ${OCI_AUTH_TYPE} == "instance_principal" ] ; then
  echo "auth:" > $OUTPUT_FILE
  echo "  authtype: instance_principal" >> $OUTPUT_FILE
fi

if [ ${OCI_AUTH_TYPE} == "user_principal" ] ; then
  if [[ ! -f ${OCI_CONFIG_FILE} ]]; then
    echo "OCI CLI configuration ${OCI_CONFIG_FILE} does not exist."
    usage
    exit 1
  fi

  SECTION_PROPS=$(read_config $OCI_CONFIG_FILE $SECTION *)
  eval $SECTION_PROPS

  # The entries user, fingerprint, key_file, tenancy and region are mandatory in the OCI CLI configuration file.
  # An empty/null value for any of the values in $OUTPUT_FILE indicates an issue with the configuration file.
  if [ -z "$region" ] || [ -z "$tenancy" ] || [ -z "$user" ] || [ -z "$key_file" ] || [ -z "$fingerprint" ]; then
    echo "One or more required entries are missing from section $SECTION in OCI CLI configuration."
    exit 1
  fi

  #create the yaml file
  echo "auth:" > $OUTPUT_FILE
  echo "  region: $region" >> $OUTPUT_FILE
  echo "  tenancy: $tenancy" >> $OUTPUT_FILE
  echo "  user: $user" >> $OUTPUT_FILE
  echo "  fingerprint: $fingerprint" >> $OUTPUT_FILE
  echo "  authtype: ${OCI_AUTH_TYPE}" >> $OUTPUT_FILE
  if [[ ! -z "$pass_phrase" ]]; then
    echo "  passphrase: $pass_phrase" >> $OUTPUT_FILE
  fi
  echo "  key: |" >> $OUTPUT_FILE
  cat $key_file | sed 's/^/    /' >> $OUTPUT_FILE
fi

# create the secret in verrazzano-install namespace
kubectl ${K8SCONTEXT} get secret $OCI_CONFIG_SECRET_NAME -n $TARGET_NS > /dev/null 2>&1
if [ $? -eq 0 ]; then
  # secret exists
  echo "Secret $OCI_CONFIG_SECRET_NAME already exists in ${TARGET_NS} namespace. Please delete that and try again."
  exit 1
fi
kubectl ${K8SCONTEXT} create secret -n $TARGET_NS  generic $OCI_CONFIG_SECRET_NAME --from-file=$OUTPUT_FILE
