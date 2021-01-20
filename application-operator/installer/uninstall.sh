#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

"$SCRIPT_DIR"/scripts/1-uninstall-vz-oam.sh
"$SCRIPT_DIR"/scripts/2-uninstall-wls-operator.sh
"$SCRIPT_DIR"/scripts/3-uninstall-coh-operator.sh
"$SCRIPT_DIR"/scripts/4-uninstall-oam-runtime.sh
