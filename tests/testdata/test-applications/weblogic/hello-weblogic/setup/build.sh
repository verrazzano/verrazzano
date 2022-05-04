#!/usr/bin/env bash
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

scriptDir="$( cd "$( dirname $0 )" && pwd )"
if [ ! -d ${scriptDir} ]; then
    echo "Unable to determine the sample directory where the application is found"
    echo "Using shell /bin/sh to determine and found ${scriptDir}"
    exit 1
fi

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

echo 'Create auxiliary image for model in image deployment'

imagetool.sh createAuxImage \
    --tag $1 \
    --wdtModel wdt_domain.yaml \
    --wdtArchive wdt_archive.zip
