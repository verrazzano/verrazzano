#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -e -o pipefail
set -xv

if [ -z "$RPM_SPEC_REPO" ] || [ -z "$VZ_VERSION" ] || [ -z "$MODULE_STREAM_VERSION" ] || [ -z "$MODULE_VERSION" ] ||
  [ -z "$TMP_BUILD_DIR" ] || [ -z "$GO_REPO_PATH" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

create_spec_repo() {
  rm -rf "${RPM_SPEC_REPO}"
  mkdir -p "${RPM_SPEC_REPO}"
  cp "${GO_REPO_PATH}/verrazzano/tools/vz/out/verrazzano-cli.spec" "${RPM_SPEC_REPO}"
  cp "${GO_REPO_PATH}/verrazzano/tools/vz/out/verrazzano-cli-${VZ_VERSION}.tar.gz" "${RPM_SPEC_REPO}"
  cd "${RPM_SPEC_REPO}"
  git init
  git add *
  git commit -m "Verrazzano CLI RPM spec file"
}

generate_module_def() {
  cat <<EOF >"${TMP_BUILD_DIR}"/verrazzano-cli-stream.yaml
document: modulemd
version: 2
data:
  name: verrazzano-cli
  stream: ${MODULE_STREAM_VERSION}
  version: ${MODULE_VERSION}
  summary: Verrazzano command-line utility
  description: >-
    The Verrazzano CLI is a command-line utility that allows Verrazzano
    operators to query and manage a Verrazzano environment.
  license:
    module:
    - UPL 1.0
    content:
    - MIT
    - ISC
    - MPL 2.0
    - UPL 1.0
    - BSD
    - Apache 2.0
  dependencies:
  - buildrequires:
      platform: [el8.7.0]
    requires:
      platform: [el8]
  profiles:
    default:
      rpms:
      - verrazzano-cli
  references:
    documentation: https://verrazzano.io/docs
  components:
    rpms:
      verrazzano-cli:
        rationale: Verrazzano command-line utility
        repository: file://${TMP_BUILD_DIR}/verrazzano-cli
        arches: [aarch64, x86_64]
EOF
}

create_spec_repo
generate_module_def
exit 0
