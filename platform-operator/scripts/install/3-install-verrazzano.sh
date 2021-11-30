#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/config.sh

CONFIG_DIR=$SCRIPT_DIR/config

platform_operator_install_message "Verrazzano"
platform_operator_install_message "Coherence Kubernetes operator"
platform_operator_install_message "WebLogic Kubernetes operator"
platform_operator_install_message "OAM Kubernetes operator"
platform_operator_install_message "Verrazzano Application Kubernetes operator"
