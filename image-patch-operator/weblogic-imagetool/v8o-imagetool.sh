#!/bin/bash
#
# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

printf "\n=====v8o-imagetool.sh has been called!=====\n\n"

echo "Environment variables:"
echo $IMAGE_REGISTRY
echo $IMAGE_REPOSITORY
echo $IMAGE_TENANCY
echo $IMAGE_NAME
echo $IMAGE_TAG
echo ${IMAGE_REGISTRY}/${IMAGE_TENANCY}/${IMAGE_REPOSITORY}:${IMAGE_TAG}
echo ""

export WLSIMG_CACHEDIR="/home/verrazzano/cache"
./imagetool/bin/imagetool.sh cache addInstaller --type wls --version 12.2.1.4.0 --path ./installers/${WEBLOGIC_INSTALLER_BINARY}
./imagetool/bin/imagetool.sh cache addInstaller --type jdk --version 8u281 --path ./installers/${JDK_INSTALLER_BINARY}
./imagetool/bin/imagetool.sh cache addInstaller --type wdt --version latest --path ./installers/${WDT_INSTALLER_BINARY}

./imagetool/bin/imagetool.sh create --tag ${IMAGE_NAME}:${IMAGE_TAG} --builder podman --jdkVersion ${JDK_INSTALLER_VERSION} --version ${WEBLOGIC_INSTALLER_VERSION}

cat /registry-creds/password | podman login $(cat /registry-creds/registry) --username $(cat /registry-creds/username) --password-stdin
podman tag ${IMAGE_NAME}:${IMAGE_TAG} ${IMAGE_REGISTRY}/${IMAGE_TENANCY}/${IMAGE_REPOSITORY}:${IMAGE_TAG}
podman images # FIXME: remove this later
podman image push ${IMAGE_REGISTRY}/${IMAGE_TENANCY}/${IMAGE_REPOSITORY}:${IMAGE_TAG}

