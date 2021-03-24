#!/bin/bash
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Add YAML boilerplate to generated CRDs - kubebuilder currently does not seem to have a way to
# add boilerplate headers to these - only to generated Go files

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
GENERATED_CRDS_DIR=${SCRIPT_DIR}/../../platform-operator/helm_config/charts/verrazzano-application-operator/crds

go run ${SCRIPT_DIR}/../../tools/fix-copyright/copyright.go -verbose $GENERATED_CRDS_DIR
