#!/bin/bash

#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

. ./init.sh

$SCRIPT_DIR/terraform destroy -var-file=$TF_VAR_nodepool_config.tfvars -auto-approve -no-color
