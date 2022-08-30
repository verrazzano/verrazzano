#!/bin/bash

ZIPFILE=$1
VZ_DEV_VERSION=$2
#TYPE=$3 # Open-source or Commercial

OPENSOURCE_EXTRACTED="verrazzano-${VZ_DEV_VERSION}-open-source-extracted"
mkdir -p $OPENSOURCE_EXTRACTED

# Extract the ZIP
tar xvf $ZIPFILE -C $OPENSOURCE_EXTRACTED

LINUX_EXTRACTED="linux-${VZ_DEV_VERSION}"
DARWIN_EXTRACTED="darwin-${VZ_DEV_VERSION}"

mkdir -p $OPENSOURCE_EXTRACTED/$LINUX_EXTRACTED
tar xvf $OPENSOURCE_EXTRACTED/verrazzano-${VZ_DEV_VERSION}-linux-amd64.tar.gz -C $OPENSOURCE_EXTRACTED/$LINUX_EXTRACTED

mkdir -p $OPENSOURCE_EXTRACTED/$DARWIN_EXTRACTED
tar xvf $OPENSOURCE_EXTRACTED/verrazzano-${VZ_DEV_VERSION}-darwin-amd64.tar.gz -C $OPENSOURCE_EXTRACTED/$DARWIN_EXTRACTED


function checkIfFileExists() {
  fileName=$1
  inputDirectory=$2
  if [ ! -e $inputDirectory/$fileName ]
  then
#    echo "FOUND ${fileName} at ${inputDirectory}"
#  else
    echo "ERROR: Missing file ${fileName} at ${inputDirectory}"
    exit 1
  fi
}

checkIfFileExists "README.MD" $OPENSOURCE_EXTRACTED
checkIfFileExists "operator.yaml" ${OPENSOURCE_EXTRACTED}
checkIfFileExists "operator.yaml.sha256" ${OPENSOURCE_EXTRACTED}

# Verify the tar files and sha256 are present for linux and darwin
checkIfFileExists "verrazzano-1.4.0-darwin-amd64.tar.gz" ${OPENSOURCE_EXTRACTED}
checkIfFileExists "verrazzano-1.4.0-darwin-amd64.tar.gz.sha256" ${OPENSOURCE_EXTRACTED}
checkIfFileExists "verrazzano-1.4.0-linux-amd64.tar.gz" ${OPENSOURCE_EXTRACTED}
checkIfFileExists "verrazzano-1.4.0-linux-amd64.tar.gz.sha256" ${OPENSOURCE_EXTRACTED}

checkIfFileExists "LICENSE" $OPENSOURCE_EXTRACTED/${LINUX_EXTRACTED}
checkIfFileExists "LICENSE" $OPENSOURCE_EXTRACTED/${DARWIN_EXTRACTED}

function verify_bin() {
  VARIANT=$1
  checkIfFileExists "bom_utils.sh" $OPENSOURCE_EXTRACTED/${VARIANT}/bin
  checkIfFileExists "vz" $OPENSOURCE_EXTRACTED/${VARIANT}/bin
  checkIfFileExists "vz-registry-image-helper.sh" $OPENSOURCE_EXTRACTED/${VARIANT}/bin
}

function verify_manifests() {
  VARIANT=$1
  checkIfFileExists "verrazzano-bom.json" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests

  checkIfFileExists "verrazzano-platform-operator.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/k8s

  #checkIfFileExists "" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/profiles

  checkIfFileExists "Chart.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator
  checkIfFileExists "NOTES.txt" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator
  checkIfFileExists "values.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator

  checkIfFileExists "clusters.verrazzano.io_verrazzanomanagedclusters.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/crds
  checkIfFileExists "install.verrazzano.io_verrazzanos.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/crds

  checkIfFileExists "clusterrole.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "clusterrolebinding.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "deployment.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "namespace.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "service.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "serviceaccount.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "validatingwebhookconfiguration.yaml" $OPENSOURCE_EXTRACTED/${VARIANT}/manifests/charts/verrazzano-platform-operator/templates
}

verify_bin $LINUX_EXTRACTED
verify_manifests $LINUX_EXTRACTED

verify_bin $DARWIN_EXTRACTED
verify_manifests $DARWIN_EXTRACTED

#sh /Users/sdosapat/src/github.com/verrazzano/verrazzano/tools/scripts/vz-registry-image-helper.sh -d -o -b $OPENSOURCE_EXTRACTED/${LINUX_EXTRACTED}/manifests/verrazzano-bom.json

function validation_cleanup() {
  # Cleaning up extracted directory
   rm -rf $OPENSOURCE_EXTRACTED
}

validation_cleanup