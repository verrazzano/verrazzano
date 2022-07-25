#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

SAVE_DIR=$(pwd)
cd $SCRIPT_DIR/../vz
echo "Running vz $@"
GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go $@ || true
cd $SAVE_DIR
