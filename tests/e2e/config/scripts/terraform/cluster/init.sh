#!/usr/bin/env bash

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
  curl https://releases.hashicorp.com/terraform/0.12.25/terraform_0.12.25_$(uname -s | tr '[:upper:]' '[:lower:]')_amd64.zip -o $SCRIPT_DIR/terraform_0.12.25.zip
  unzip $SCRIPT_DIR/terraform_0.12.25.zip -d $SCRIPT_DIR/
  rm $SCRIPT_DIR/terraform_0.12.25.zip
fi

if [ ! -f $SCRIPT_DIR/terraform ];
then
  echo "terraform is required and can not be found."
  exit 1
fi

set -e

$SCRIPT_DIR/terraform init -no-color -reconfigure \
  --backend-config "key=${TF_VAR_state_name}" \
  
$SCRIPT_DIR/terraform plan -var-file=$TF_VAR_nodepool_config.tfvars -no-color
