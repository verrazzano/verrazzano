#!/bin/bash

#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage {
    echo
    echo "usage: $0 [-o operation] [-c compartment_ocid] [-s subdomain_name] "
    echo "  -o operation               'create' or 'delete'. Optional.  Defaults to 'create'."
    echo "  -c compartment_ocid        Compartment OCID. Optional.  Defaults to TIBURON-DEV compartment OCID."
    echo "  -s submdomain_name         subdomain prefix for v8o.oracledx.com. Required."
    echo "  -h                         Help"
    echo
    exit 1
}

SUBDOMAIN_NAME=""
COMPARTMENT_OCID="${TF_VAR_compartment_id}"
OPERATION="create"

while getopts o:c:s:h flag
do
    case "${flag}" in
        o) OPERATION=${OPTARG};;
        c) COMPARTMENT_OCID=${OPTARG};;
        s) SUBDOMAIN_NAME=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

if [ -z "${SUBDOMAIN_NAME}" ] ; then
    echo "subdomain name must be set!"
    exit 1
fi

set -o pipefail
ZONE_NAME="${SUBDOMAIN_NAME}.v8o.oracledx.com"

zone_ocid=""
if [ $OPERATION == "create" ]; then
  # the installation will require the "patch" command, so will install it now.  If it's already installed yum should
  # exit
  sudo yum -y install patch >/dev/null 2>&1
  zone_ocid=$(oci dns zone create -c ${COMPARTMENT_OCID} --name ${ZONE_NAME} --zone-type PRIMARY | jq -r ".data | .[\"id\"]")
elif [ $OPERATION == "delete" ]; then
  oci dns zone delete --zone-name-or-id ${ZONE_NAME} --force
  if [ $? -ne 0 ]; then
    echo "DNS zone deletion failed on first try. Retrying once."
    oci dns zone delete --zone-name-or-id ${ZONE_NAME} --force
  fi
else
  echo "Unknown operation: ${OPERATION}"
  usage
fi

status_code=$?
if [ ${status_code:-1} -eq 0 ]; then
  # OCI CLI query succeeded
  if [ $OPERATION == "create" ]; then
    echo $zone_ocid
  else
    exit 0
  fi
else
  # OCI CLI generated an error exit code
  echo "Error invoking OCI CLI to perform DNS zone operation"
  exit 1
fi

