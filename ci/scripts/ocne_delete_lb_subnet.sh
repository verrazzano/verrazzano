#!/bin/bash

#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# needs env vars: VCN_OCID, TF_VAR_compartment_id, TF_VAR_prefix

get_security_list_id() {
  n=0
  while [ $n -le 5 ] && [ -z "${id}" ]; do
    id=$(oci network security-list list --display-name "${TF_VAR_prefix}-lb-subnet" --compartment-id "${TF_VAR_compartment_id}" --vcn-id "${VCN_OCID}" --query 'data[0]."id"' --raw-output)
    n=$((n+1))
    sleep 2
  done
  echo "${id}"
}

get_subnet_id() {
  n=0
  while [ $n -le 5 ] && [ -z "${id}" ]; do
    id=$(oci network subnet list --display-name "${TF_VAR_prefix}-lb-subnet" --compartment-id "${TF_VAR_compartment_id}" --vcn-id "${VCN_OCID}" --query 'data[0]."id"' --raw-output)
    n=$((n+1))
    sleep 2
  done
  echo "${id}"
}

delete_security_list() {
  oci network security-list delete --force --security-list-id "${security_list_id}"
}

delete_subnet() {
  oci network subnet delete --force --subnet-id "${lb_subnet_id}"
}

lb_subnet_id=$(get_subnet_id)
if [ -z "${lb_subnet_id}" ]; then
  echo "Failed to find subnet"
else
  echo "deleting subnet ${lb_subnet_id}"
  delete_subnet
fi
  
security_list_id=$(get_security_list_id)
if [ -z "${security_list_id}" ]; then
    echo "Failed to find security list"
else
    echo "deleting security list ${security_list_id}"
    delete_security_list
fi

