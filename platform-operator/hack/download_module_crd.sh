#!/bin/bash
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Add Module Operator CRD to verrazzano-platform-operator helm chart

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
wget https://raw.githubusercontent.com/verrazzano/verrazzano-modules/main/module-operator/manifests/charts/operators/verrazzano-module-operator/crds/platform.verrazzano.io_modules.yaml \
  -O $SCRIPT_DIR/../../platform-operator/helm_config/charts/verrazzano-platform-operator/crds/platform.verrazzano.io_modules.yaml
