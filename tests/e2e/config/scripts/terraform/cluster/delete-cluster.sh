#!/usr/bin/env bash

. ./init.sh

$SCRIPT_DIR/terraform destroy -var-file=$TF_VAR_nodepool_config.tfvars -auto-approve -no-color
