#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
set -e -o pipefail
set -xv

if [ -z "$JENKINS_URL" ] || [ -z "$BUILD_OS" ] || [ -z "$MODULE_BUILD_DIR" ] || [ -z "$MODULE_REPO_ARCHIVE_DIR" ] ||
  [ -z "$MODULE_STREAM_VERSION" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

cat <<EOF >"$MODULE_BUILD_DIR/verrazzano-${BUILD_OS}.repo"
[${BUILD_OS}_verrazzano]
name=Oracle Verrazzano (\$basearch)
baseurl=${MODULE_REPO_ARCHIVE_DIR}/results
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-oracle
gpgcheck=0
enabled=1
EOF

sudo mv "$MODULE_BUILD_DIR/verrazzano-${BUILD_OS}.repo" "/etc/yum.repos.d/verrazzano-${BUILD_OS}.repo"
sudo dnf module list
sudo dnf module install -y "verrazzano-cli:${MODULE_STREAM_VERSION}/default"
vz version
sudo dnf remove -y verrazzano-cli
