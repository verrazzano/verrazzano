#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install
UNINSTALL_DIR=$SCRIPT_DIR/..
MANIFESTS_DIR=$SCRIPT_DIR/../../../thirdparty/manifests

. $INSTALL_DIR/common.sh
. $INSTALL_DIR/config.sh
. $UNINSTALL_DIR/uninstall-utils.sh

set -o pipefail

