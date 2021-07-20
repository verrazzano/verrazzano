#!/bin/bash
#
# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Log into the registry
if ! (cat /registry-creds/password | podman login $(cat /registry-creds/registry) --username $(cat /registry-creds/username) --password-stdin); then
  echo "Podman login failed."
  exit 1
fi

# Add installers to the Image Tool cache
export WLSIMG_CACHEDIR="/home/verrazzano/cache"
if ! ./imagetool/bin/imagetool.sh cache addInstaller --type wls --version ${WEBLOGIC_INSTALLER_VERSION} --path ./installers/${WEBLOGIC_INSTALLER_BINARY}; then
  echo "Adding WebLogic installer to Image Tool cache failed."
  exit 1
fi
if ! ./imagetool/bin/imagetool.sh cache addInstaller --type jdk --version ${JDK_INSTALLER_VERSION} --path ./installers/${JDK_INSTALLER_BINARY}; then
  echo "Adding JDK installer to Image Tool cache failed."
  exit 1
fi
if ! ./imagetool/bin/imagetool.sh cache addInstaller --type wdt --version ${WDT_INSTALLER_VERSION} --path ./installers/${WDT_INSTALLER_BINARY}; then
  echo "Adding WebLogic Deploy Tooling installer to Image Tool cache failed."
  exit 1
fi

# Create the image
if ! ./imagetool/bin/imagetool.sh create --tag ${IMAGE_NAME}:${IMAGE_TAG} --builder podman --jdkVersion ${JDK_INSTALLER_VERSION} --version ${WEBLOGIC_INSTALLER_VERSION}; then
  echo "Creating the image using WebLogic Image Tool failed."
  exit 1
fi

# Tag and push the image to the registry
if ! podman tag ${IMAGE_NAME}:${IMAGE_TAG} ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}/${IMAGE_NAME}:${IMAGE_TAG}; then
  echo "Tagging image failed."
  exit 1
fi
if ! podman image push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}/${IMAGE_NAME}:${IMAGE_TAG}; then
  echo "Pushing image failed."
  exit 1
fi

echo "Successfully pushed image!"
