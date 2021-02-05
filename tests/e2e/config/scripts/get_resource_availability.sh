#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Please specify the service and limit name"
    exit 1
fi

service_name=$1
limit_name=$2

if [ -z "${TF_VAR_compartment_id}" ] ; then
    echo "TF_VAR_compartment_id env var must be set!'"
    exit 1
fi

if [ -z "${TF_VAR_region}" ] ; then
    echo "TF_VAR_region env var must be set!'"
    exit 1
fi

set -o pipefail

count=$(oci limits resource-availability get -c ${TF_VAR_compartment_id} --region ${TF_VAR_region} --service-name ${service_name} --limit-name ${limit_name} | jq ".data | .[\"available\"]")

status_code=$?
if [ ${status_code:-1} -eq 0 ]; then
  # OCI CLI query succeeded
  echo $count
else
  # OCI CLI generated an error exit code
  echo "Error invoking OCI CLI to obtain resource availability of VCNs"
  exit 1
fi

