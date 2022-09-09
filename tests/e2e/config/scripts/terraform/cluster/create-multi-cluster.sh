#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

CLUSTER_INDEX=${1:-1}
CLUSTER_NAME_PREFIX=${2:-""}
. ./init.sh

$SCRIPT_DIR/terraform init -no-color

workspace=cluster-${CLUSTER_INDEX}
echo "Creating Terraform workspace: $workspace"
$SCRIPT_DIR/terraform workspace new $workspace -no-color

$SCRIPT_DIR/terraform plan -var-file=$TF_VAR_nodepool_config.tfvars -var-file=$TF_VAR_region.tfvars -no-color

set -o pipefail

# retry 3 times, 30 seconds apart
tries=0
MAX_TRIES=3
while true; do
   tries=$((tries+1))
   echo "terraform apply iteration ${tries}"
   $SCRIPT_DIR/terraform apply -var-file=$TF_VAR_nodepool_config.tfvars -var-file=$TF_VAR_region.tfvars -auto-approve -no-color && break
   if [ "$tries" -ge "$MAX_TRIES" ];
   then
      echo "Terraform apply tries exceeded.  Cluster creation has failed!"
      break
   fi
   sleep 30
done

if [ "$tries" -ge "$MAX_TRIES" ];
then
  exit 1
fi

echo "Updating OKE private_workers_seclist to allow pub_lb_subnet access to workers"

# the script would return 0 even if it fails to update OKE private_workers_seclist
# because the OKE still could work if it didn't hit the rate limiting

# find vcn id
VCN_ID=$(oci network vcn list \
  --compartment-id "${TF_VAR_compartment_id}" \
  --display-name "${TF_VAR_label_prefix}-${CLUSTER_NAME_PREFIX}-${CLUSTER_INDEX}-vcn" \
  | jq -r '.data[0].id')
if [ -z "$VCN_ID" ]; then
    echo "Failed to get the id for OKE cluster vcn ${TF_VAR_label_prefix}-${CLUSTER_NAME_PREFIX}-${CLUSTER_INDEX}-vcn"
    exit 0
fi

# find private_workers_seclist id
NSG_ID=$(oci network nsg list \
  --compartment-id "${TF_VAR_compartment_id}" \
  --display-name "${TF_VAR_label_prefix}-private-workers" \
  --vcn-id "${VCN_ID}" \
  | jq -r '.data[0].id')

if [ -z "$NSG_ID" ]; then
    echo "Failed to get the id for NSG ${TF_VAR_label_prefix}-workers"
    exit 0
fi

# find pub_lb_subnet CIDR
LB_SUBNET_CIDR=$(oci network subnet list \
  --compartment-id "${TF_VAR_compartment_id}" \
  --display-name "${TF_VAR_label_prefix}-pub_lb" \
  --vcn-id "${VCN_ID}" \
  | jq -r '.data[0]."cidr-block"')

if [ -z "$LB_SUBNET_CIDR" ]; then
    echo "Failed to get the cidr-block for subnet ${TF_VAR_label_prefix}-pub_lb"
    exit 0
fi

# add pub_lb_subnet ingress-security-rule
cat <<EOF > new.ingress-security-rules-${CLUSTER_INDEX}.json
[{"description": "allow pub_lb_subnet access to workers","is-stateless": false,"direction": "INGRESS","protocol": "6","source": "$LB_SUBNET_CIDR","tcp-options": {"destination-port-range": {"max": 32767,"min": 30000}}},{"description": "allow pub_lb_subnet health check access to workers","is-stateless": false,"direction": "INGRESS","protocol": "6","source": "$LB_SUBNET_CIDR","tcp-options": {"destination-port-range": {"max": 10256,"min": 10256}}}]
EOF

# update private_workers_seclist
oci network nsg rules add --nsg-id "${NSG_ID}" --security-rules "file://${PWD}/new.ingress-security-rules-${CLUSTER_INDEX}.json"
if [ $? -eq 0 ]; then
  echo "Updated the OKE private_workers_seclist"
else
  echo "Failed to update the OKE private_workers_seclist"
fi
