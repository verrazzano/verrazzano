#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
OAM_COMP_FILE=$1
IMG_LIST_FILE=$2

if [ -z "${OAM_COMP_FILE}" ] ; then
    echo "The script requires OAM component file defining the image(s) for the example"
    exit 1
fi

if [ -z "${IMG_LIST_FILE}" ] ; then
    echo "The script requires a file to append the example image(s)"
    exit 1
fi

cat ${OAM_COMP_FILE} | grep image: | grep example |\tr -s '[[:space:]]' '\n' |\sort |\uniq | grep verrazzano | grep / | cut -d/ -f2- | tr -d '"' >> ${IMG_LIST_FILE} || exit 1
