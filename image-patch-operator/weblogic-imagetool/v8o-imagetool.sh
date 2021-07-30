#!/bin/bash
#
# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# This script will fail if any line fails
set -e

# Log into the registry
cat /registry-creds/password | podman login $(cat /registry-creds/registry) --username $(cat /registry-creds/username) --password-stdin

# Add installers to the imagetool cache
export WLSIMG_CACHEDIR="/home/verrazzano/cache"
./imagetool/bin/imagetool.sh cache addInstaller --type wls --version ${WEBLOGIC_INSTALLER_VERSION} --path /installers/${WEBLOGIC_INSTALLER_BINARY}
./imagetool/bin/imagetool.sh cache addInstaller --type jdk --version ${JDK_INSTALLER_VERSION} --path /installers/${JDK_INSTALLER_BINARY}
./imagetool/bin/imagetool.sh cache addInstaller --type wdt --version ${WDT_INSTALLER_VERSION} --path /installers/${WDT_INSTALLER_BINARY}

# Create the image
./imagetool/bin/imagetool.sh create --tag ${IMAGE_NAME}:${IMAGE_TAG} --builder podman --jdkVersion ${JDK_INSTALLER_VERSION} --version ${WEBLOGIC_INSTALLER_VERSION}

# Tag and push the image to the registry

echo time=$(date +"%Y-%m-%dT%TZ") podman tag
podman tag ${IMAGE_NAME}:${IMAGE_TAG} ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}/${IMAGE_NAME}:${IMAGE_TAG}

echo time=$(date +"%Y-%m-%dT%TZ") podman image push
podman image push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}/${IMAGE_NAME}:${IMAGE_TAG}
