#!/bin/bash

ZIPFILE=$1
VZ_DEV_VERSION=$2
TYPE=$3 # Open-source or Commercial



if [ $TYPE == "Commercial" ]
  then
    echo "Provided type is Commercial"
    TYPE_EXTRACTED="verrazzano-${VZ_DEV_VERSION}-commercial-extracted"
    mkdir -p $TYPE_EXTRACTED
#    unzip $ZIPFILE -d $TYPE_EXTRACTED
  else
    echo "Provided OpenSource"
    TYPE_EXTRACTED="verrazzano-${VZ_DEV_VERSION}-open-source-extracted"
    mkdir -p $TYPE_EXTRACTED
#    unzip $ZIPFILE -d $TYPE_EXTRACTED

    LINUX_EXTRACTED="linux-${VZ_DEV_VERSION}"
    DARWIN_EXTRACTED="darwin-${VZ_DEV_VERSION}"

    mkdir -p $TYPE_EXTRACTED/$LINUX_EXTRACTED
    tar xvf $TYPE_EXTRACTED/verrazzano-${VZ_DEV_VERSION}-linux-amd64.tar.gz -C $TYPE_EXTRACTED/$LINUX_EXTRACTED

    mkdir -p $TYPE_EXTRACTED/$DARWIN_EXTRACTED
    tar xvf $TYPE_EXTRACTED/verrazzano-${VZ_DEV_VERSION}-darwin-amd64.tar.gz -C $TYPE_EXTRACTED/$DARWIN_EXTRACTED
fi

ls $TYPE_EXTRACTED

function checkIfFileExists() {
  fileName=$1
  inputDirectory=$2
  if [ ! -e $inputDirectory/$fileName ]
  then
    echo "ERROR: Missing file ${fileName} at ${inputDirectory}"
    ls $inputDirectory
#    exit 1 #TODO enable exit code
#  else
#    echo "FOUND ${fileName} at ${inputDirectory}"
  fi
}

function verify_commercial_contents() {
  checkIfFileExists "LICENSE" $TYPE_EXTRACTED
  checkIfFileExists "README.MD" $TYPE_EXTRACTED
}

function verify_commercial_bin() {
  DIR=$1
  checkIfFileExists "bom_utils.sh" $DIR/bin
  checkIfFileExists "vz-registry-image-helper.sh" $DIR/bin

  checkIfFileExists "vz" "${DIR}/bin/darwin-amd64"
  checkIfFileExists "vz" "${DIR}/bin/darwin-arm64"
  checkIfFileExists "vz" "${DIR}/bin/linux-amd64"
  checkIfFileExists "vz" "${DIR}/bin/linux-arm64"
}

function verify_commercial_images() {
    DIR=$1
    # Verify the images count to some value
    echo "Count of all tar images are: $(ls $DIR/images | wc -l)"
}

function verify_common_manifests() {
  DIR=$1

  # Verify BOM
  checkIfFileExists "verrazzano-bom.json" $DIR/manifests

  # Verify K8S
  checkIfFileExists "verrazzano-platform-operator.yaml" $DIR/manifests/k8s

  #checkIfFileExists "" $DIR/manifests/profiles

  # Verify verrazzano-platform-operator
  checkIfFileExists "Chart.yaml" $DIR/manifests/charts/verrazzano-platform-operator
  checkIfFileExists "NOTES.txt" $DIR/manifests/charts/verrazzano-platform-operator
  checkIfFileExists "values.yaml" $DIR/manifests/charts/verrazzano-platform-operator

  # Verify verrazzano-platform-operator/crds
  checkIfFileExists "clusters.verrazzano.io_verrazzanomanagedclusters.yaml" $DIR/manifests/charts/verrazzano-platform-operator/crds
  checkIfFileExists "install.verrazzano.io_verrazzanos.yaml" $DIR/manifests/charts/verrazzano-platform-operator/crds

  # Verify verrazzano-platform-operator/templates
  checkIfFileExists "clusterrole.yaml" $DIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "clusterrolebinding.yaml" $DIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "deployment.yaml" $DIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "namespace.yaml" $DIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "service.yaml" $DIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "serviceaccount.yaml" $DIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "validatingwebhookconfiguration.yaml" $DIR/manifests/charts/verrazzano-platform-operator/templates
}

function verify_opensource_contents() {
  checkIfFileExists "README.MD" $TYPE_EXTRACTED
  checkIfFileExists "operator.yaml" ${TYPE_EXTRACTED}
  checkIfFileExists "operator.yaml.sha256" ${TYPE_EXTRACTED}

  # Verify the tar files and sha256 are present for linux and darwin
  checkIfFileExists "verrazzano-1.4.0-darwin-amd64.tar.gz" ${TYPE_EXTRACTED}
  checkIfFileExists "verrazzano-1.4.0-darwin-amd64.tar.gz.sha256" ${TYPE_EXTRACTED}
  checkIfFileExists "verrazzano-1.4.0-linux-amd64.tar.gz" ${TYPE_EXTRACTED}
  checkIfFileExists "verrazzano-1.4.0-linux-amd64.tar.gz.sha256" ${TYPE_EXTRACTED}

  checkIfFileExists "LICENSE" $TYPE_EXTRACTED/${LINUX_EXTRACTED}
  checkIfFileExists "LICENSE" $TYPE_EXTRACTED/${DARWIN_EXTRACTED}
}

function verify_opensource_bin() {
  DIR=$1
  checkIfFileExists "bom_utils.sh" $DIR/bin
  checkIfFileExists "vz" $DIR/bin
  checkIfFileExists "vz-registry-image-helper.sh" $DIR/bin
}

if [ $TYPE == "Commercial" ]
  then
    echo "Provided type is Commercial"
    verify_commercial_contents
    verify_commercial_bin $TYPE_EXTRACTED
    verify_commercial_images $TYPE_EXTRACTED
    verify_common_manifests $TYPE_EXTRACTED
  else
    echo "In OpenSource"
    ls $TYPE_EXTRACTED/$LINUX_EXTRACTED # TODO remove
    ls $TYPE_EXTRACTED/$DARWIN_EXTRACTED #TODO remove

    verify_opensource_contents
    verify_opensource_bin $TYPE_EXTRACTED/$LINUX_EXTRACTED
    verify_common_manifests $TYPE_EXTRACTED/$LINUX_EXTRACTED
    verify_opensource_bin $TYPE_EXTRACTED/$DARWIN_EXTRACTED
    verify_common_manifests $TYPE_EXTRACTED/$DARWIN_EXTRACTED
fi

# TODO verify BOM content for open source

function validation_cleanup() {
  # Cleaning up extracted directory
   rm -rf $TYPE_EXTRACTED
}

#validation_cleanup