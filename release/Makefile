# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

SCRIPT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

# If RELEASE_VERSION is not set, use .verrazzano-development-version
RELEASE_VERSION ?= $(shell grep verrazzano-development-version ${SCRIPT_DIR}/../.verrazzano-development-version | sed -e 's/verrazzano-development-version=//')

# File containing list of images
ifndef VZ_IMAGES_LIST
$(error Environment variable VZ_IMAGES_LIST, defining the components and corresponding images, is not defined)
endif

# File containing the source repository URLs
ifndef VZ_REPO_URLS
$(error Environment variable VZ_REPO_URLS, containing the repository URLs for the Verrazzano components, is not defined)
endif

# Define default values for the directory to download the source and create the archive
OUTPUT_DIR ?= ${SCRIPT_DIR}/out/source
ARCHIVE_DIR ?= ${SCRIPT_DIR}/out/archive

.PHONY: source-archive
source-archive:
	# Dry run to validate VZ_REPO_URLS contains all the required files
	${SCRIPT_DIR}/scripts/download_source_prt.sh -i ${VZ_IMAGES_LIST} -r ${VZ_REPO_URLS} -s ${OUTPUT_DIR} -d true; \

	# Download the source files
	${SCRIPT_DIR}/scripts/download_source_prt.sh -i ${VZ_IMAGES_LIST} -r ${VZ_REPO_URLS} -s ${OUTPUT_DIR}; \

	# Create the archive from the downloaded source
	${SCRIPT_DIR}/scripts/archive_source_prt.sh ${OUTPUT_DIR} ${ARCHIVE_DIR} ${RELEASE_VERSION}; \

