#!/usr/bin/env bash
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

WEBLOGIC_DOMAIN_MODE=
IMAGE=

scriptDir="$( cd "$( dirname $0 )" && pwd )"
if [ ! -d ${scriptDir} ]; then
    echo "Unable to determine the sample directory where the application is found"
    echo "Using shell /bin/sh to determine and found ${scriptDir}"
    exit 1
fi

function usage() {
    echo """
Script to create an image in WebLogic ImageTool.

Usage:

$0 -d <weblogic_domain_mode> -i <image_name>

Options:
 -d <weblogic_domain_mode>  WebLogic domain mode (model in image or auxiliary image)
 -i <image_name>            Docker image name
 -h                         Display help usage
"""
exit 0
}

exit_error() {
  usage
  exit 1
}

setup() {
  echo 'Build the WDT archive...'
  rm wdt_archive.zip
  rm weblogic-deploy.zip
  mkdir -p ./wlsdeploy/applications
  cp ../target/hello.war ./wlsdeploy/applications
  zip -r wdt_archive.zip wlsdeploy
  rm -r wlsdeploy

  echo 'Download WDT...'
  if [ -f weblogic-deploy.zip ]; then
      echo 'Using existing weblogic-deploy.zip...'
  else
      echo 'Downloading weblogic-deploy.zip...'
      wget https://github.com/oracle/weblogic-deploy-tooling/releases/download/release-2.1.0/weblogic-deploy.zip
  fi

  echo 'Download WebLogic Image Tool...'
  if [ -f imagetool.zip ]; then
      echo 'Using existing imagetool.zip...'
  else
      echo 'Downloading imagetool.zip...'
      wget https://github.com/oracle/weblogic-image-tool/releases/download/release-1.10.0/imagetool.zip
      unzip imagetool.zip
  fi

  export PATH=`pwd`/imagetool/bin:$PATH

  echo 'Add installers to Image Tool cache...'
  imagetool.sh cache addInstaller --type wdt --version latest --path weblogic-deploy.zip --force

  if [[ $WEBLOGIC_DOMAIN_MODE == "a" ]]; then
    echo 'Create auxiliary image for model in image deployment'
    imagetool.sh createAuxImage \
        --tag $IMAGE \
        --wdtModel wdt_domain.yaml \
        --wdtArchive wdt_archive.zip
  elif [[ $WEBLOGIC_DOMAIN_MODE == "m" ]]; then
    imagetool.sh cache addInstaller --type jdk --version 8u261 --path ${JDK8_BUNDLE}
    imagetool.sh cache addInstaller --type wls --version 12.2.1.4.0 --path ${WEBLOGIC_BUNDLE}
    echo 'Create image for model in image deployment'
    imagetool.sh create \
        --tag $IMAGE \
        --version 12.2.1.4.0 \
        --jdkVersion 8u261 \
        --fromImage container-registry.oracle.com/os/oraclelinux:7-slim \
        --wdtModel wdt_domain.yaml \
        --wdtArchive wdt_archive.zip \
        --wdtModelOnly
  fi
}

if [[ $# -ne 0 ]]; then
  while getopts 'hd:i:' OPTION; do
  case $OPTION in
  i)
    IMAGE=$OPTARG
    ;;
  d)
    WEBLOGIC_DOMAIN_MODE=$OPTARG
    if [[ $WEBLOGIC_DOMAIN_MODE != "m" && $WEBLOGIC_DOMAIN_MODE != "a" ]]; then
      exit_error
    fi
    ;;
  h)
    usage
    ;;
  ?)
    exit_error
    ;;
  esac
  done
  shift $(($OPTIND - 1))
else
  exit_error
fi

# Mandatory arguments
if [ ! "$IMAGE" ] || [ ! "$WEBLOGIC_DOMAIN_MODE" ]; then
  echo "Options -i, -d and their arguments must be provided"
  exit_error
fi

if [[ $WEBLOGIC_DOMAIN_MODE == "m" ]]; then
  if [[ -z "$JDK8_BUNDLE" ]] || [[ -z "$WEBLOGIC_BUNDLE" ]] ; then
    echo "JDK8_BUNDLE and WEBLOGIC_BUNDLE must be defined"
    exit 1
  fi
fi

setup