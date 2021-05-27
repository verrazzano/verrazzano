#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

for REQUIRED_ENV_VAR in $(cat ${SCRIPT_DIR}/required-env-vars); do
  if [ -z "${!REQUIRED_ENV_VAR}" ] ; then
    echo "${REQUIRED_ENV_VAR} not set"
    exit 1
  fi
done

#download terraform
if [ ! -f $SCRIPT_DIR/terraform ];
then
  curl https://releases.hashicorp.com/terraform/0.14.10/terraform_0.14.10_$(uname -s | tr '[:upper:]' '[:lower:]')_amd64.zip -o $SCRIPT_DIR/terraform_0.14.10.zip
  unzip $SCRIPT_DIR/terraform_0.14.10.zip -d $SCRIPT_DIR/
  rm $SCRIPT_DIR/terraform_0.14.10.zip
fi

if [ ! -f $SCRIPT_DIR/terraform ];
then
  echo "terraform is required and can not be found."
  exit 1
fi

set -e

$SCRIPT_DIR/terraform init -no-color -reconfigure

$SCRIPT_DIR/terraform plan -var-file=$TF_VAR_nodepool_config.tfvars -var-file=$TF_VAR_region.tfvars -no-color
