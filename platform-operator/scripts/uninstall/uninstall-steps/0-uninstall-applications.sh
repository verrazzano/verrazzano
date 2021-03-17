#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
UNINSTALL_DIR=$SCRIPT_DIR/..

. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

# Delete all of the OAM ApplicationConfiguration resources in all namespaces.
function delete_oam_applications_configurations {
  delete_k8s_resource_from_all_namespaces applicationconfigurations.core.oam.dev
}

# Delete all of the OAM Component resources in all namespaces.
function delete_oam_components {
  delete_k8s_resource_from_all_namespaces components.core.oam.dev
}

action "Deleting OAM application configurations" delete_oam_applications_configurations || exit 1
action "Deleting OAM components" delete_oam_components || exit 1
