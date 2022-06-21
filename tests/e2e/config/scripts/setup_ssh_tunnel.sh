#!/bin/bash

#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "TF_VAR_api_private_key_path" ] ; then
    echo "TF_VAR_api_private_key_path env var must be set!"
    exit 1
fi
if [ -z "TF_VAR_compartment_id" ] ; then
    echo "TF_VAR_compartment_id env var must be set!"
    exit 1
fi
if [ -z "TF_VAR_label_prefix" ] ; then
    echo "TF_VAR_label_prefix env var must be set!"
    exit 1
fi
if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi

# install sshuttle
sudo yum -y install oracle-epel-release-el7
sudo yum -y install sshuttle
if [ $? -ne 0 ]; then
  echo "Failed to install sshuttle."
  exit 1
fi

# find the CIDR for the VPN
VCN_CIDR=$(oci network vcn list \
  --compartment-id "${TF_VAR_compartment_id}" \
  --display-name "${TF_VAR_label_prefix}-oke-vcn" \
  --lifecycle-state AVAILABLE \
  | jq -r '.data[0]."cidr-block"')

if [ -z "VCN_CIDR" ]; then
    echo "Failed to get the CIDR for VCN ${TF_VAR_label_prefix}-oke-vcn"
    exit 1
fi

# find bastion compute instance id
BASTION_ID=$(oci compute instance list \
  --compartment-id "${TF_VAR_compartment_id}" \
  --display-name "${TF_VAR_label_prefix}-bastion" \
  --lifecycle-state RUNNING \
  | jq -r '.data[0]."id"')

if [ -z "$BASTION_ID" ]; then
    echo "Failed to get the OCID for compute instance ${TF_VAR_label_prefix}-bastion"
    exit 1
fi

# find public IP for the bastion compute instance
BASTION_IP=$(oci compute instance list-vnics \
  --compartment-id "${TF_VAR_compartment_id}" \
  --instance-id "${BASTION_ID}" \
  | jq -r '.data[0]."public-ip"')

if [ -z "$BASTION_IP" ]; then
    echo "Failed to get the public IP for compute instance ${TF_VAR_label_prefix}-bastion"
    exit 1
fi

# run sshuttle
sshuttle -r opc@$BASTION_IP $VCN_CIDR --ssh-cmd 'ssh -o StrictHostKeyChecking=no -i '${OPC_USER_KEY_FILE}'' --daemon
if [ $? -ne 0 ]; then
  echo "Failed to ssh tunnel to the bastion host ${TF_VAR_label_prefix}-bastion at ${BASTION_IP}"
  exit 1
fi
