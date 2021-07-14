#!/bin/bash
#
# Copyright (C) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

printf "\n=====v8o-imagetool.sh has been called!=====\n\n"

# TODO: move imagetool cache addInstaller here

cat /registry-cred/password | podman login $(cat /registry-creds/registry) --username $(cat /registry-cred/username) --password-stdin

./imagetool/bin/imagetool.sh create --tag ${IMAGE_NAME}:${IMAGE_TAG} --builder podman --jdkVersion ${JDK_INSTALLER_VERSION} --version ${WEBLOGIC_INSTALLER_VERSION}

podman image push ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}/${IMAGE_NAME}:${IMAGE_TAG}
