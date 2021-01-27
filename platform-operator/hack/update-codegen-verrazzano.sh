#!/bin/bash
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

set -o errexit
set -o nounset
set -o pipefail

CODEGEN_PATH=k8s.io/code-generator

SCRIPT_ROOT=$(dirname $0)/..
echo "script_root = ${SCRIPT_ROOT}"

# Obtain k8s.io/code-generator version
codeGenVer=$(grep "code-generator" go.mod | awk '{print $2}')

CODEGEN_PKG=${CODEGEN_PKG:-${GOPATH}/pkg/mod/${CODEGEN_PATH}@${codeGenVer}}
echo "codegen_pkg = ${CODEGEN_PKG}"
chmod +x ${CODEGEN_PKG}/generate-groups.sh

GENERATED_CLIENT_DIR=$SCRIPT_ROOT/client
echo Remove $GENERATED_CLIENT_DIR dir if exist
rm -rf $GENERATED_CLIENT_DIR

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
${CODEGEN_PKG}/generate-groups.sh "client" \
  github.com/verrazzano/verrazzano/platform-operator/client github.com/verrazzano/verrazzano/platform-operator/api \
  verrazzano:v1alpha1 \
  --output-base "${GOPATH}/src" \
  --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt
