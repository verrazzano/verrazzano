#!/bin/bash

ROOTDIR=$1
VZ_DEV_VERSION=$2
TYPE=$3 # lite or full bundle

VZ_DIR="verrazzano-${VZ_DEV_VERSION}"

if [ $TYPE == "Lite" ]
  then
    echo "Provided Lite bundle"
    LINUX_NAME="verrazzano-${VZ_DEV_VERSION}-linux-amd64.tar.gz"
    LINUX_EXTRACTED="linux-${VZ_DEV_VERSION}"
    DARWIN_NAME="verrazzano-${VZ_DEV_VERSION}-darwin-amd64.tar.gz"
    DARWIN_EXTRACTED="darwin-${VZ_DEV_VERSION}"

    mkdir -p $LINUX_EXTRACTED
    mkdir -p $DARWIN_EXTRACTED

    tar xvf $LINUX_NAME -C $LINUX_EXTRACTED
    tar xvf $DARWIN_NAME -C $DARWIN_EXTRACTED
fi

function checkIfFileExists() {
  fileName=$1
  inputDirectory=$2
  if [ ! -e $inputDirectory/$fileName ]
  then
    echo "ERROR: Missing file ${fileName} at ${inputDirectory}"
#    exit 1 #TODO enable exit code
  else
    echo "FOUND ${fileName} at ${inputDirectory}" #TODO cleanup
  fi
}

function verify_full_contents() {
  INPUTDIR=$1
  checkIfFileExists "LICENSE" $INPUTDIR
  checkIfFileExists "README.md" $INPUTDIR
}

function verify_full_bin() {
  INPUTDIR=$1
  checkIfFileExists "bom_utils.sh" $INPUTDIR/bin
  checkIfFileExists "vz-registry-image-helper.sh" $INPUTDIR/bin

  checkIfFileExists "vz" "${INPUTDIR}/bin/darwin-amd64"
  checkIfFileExists "vz" "${INPUTDIR}/bin/darwin-arm64"
  checkIfFileExists "vz" "${INPUTDIR}/bin/linux-amd64"
  checkIfFileExists "vz" "${INPUTDIR}/bin/linux-arm64"
}

function verify_full_images() {
    INPUTDIR=$1
    echo "Count of all tar images are: $(ls $INPUTDIR/images | wc -l)" #TODO verify images count
}

function verify_common_manifests() {
  INPUTDIR=$1

  # Verify BOM
  checkIfFileExists "verrazzano-bom.json" $INPUTDIR/manifests

  # Verify K8S
  checkIfFileExists "verrazzano-platform-operator.yaml" $INPUTDIR/manifests/k8s

  #checkIfFileExists "" $INPUTDIR/manifests/profiles

  # Verify verrazzano-platform-operator
  checkIfFileExists "Chart.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator
  checkIfFileExists "NOTES.txt" $INPUTDIR/manifests/charts/verrazzano-platform-operator
  checkIfFileExists "values.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator

  # Verify verrazzano-platform-operator/crds
  checkIfFileExists "clusters.verrazzano.io_verrazzanomanagedclusters.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/crds
  checkIfFileExists "install.verrazzano.io_verrazzanos.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/crds

  # Verify verrazzano-platform-operator/templates
  checkIfFileExists "clusterrole.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "clusterrolebinding.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "deployment.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "namespace.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "service.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "serviceaccount.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/templates
  checkIfFileExists "validatingwebhookconfiguration.yaml" $INPUTDIR/manifests/charts/verrazzano-platform-operator/templates
}

function verify_lite_contents() {
  INPUTDIR=$1
  checkIfFileExists "operator.yaml" ${INPUTDIR}
  checkIfFileExists "operator.yaml.sha256" ${INPUTDIR}

  # Verify the tar files and sha256 are present for linux and darwin
  checkIfFileExists "verrazzano-${VZ_DEV_VERSION}-darwin-amd64.tar.gz" ${INPUTDIR}
  checkIfFileExists "verrazzano-${VZ_DEV_VERSION}-darwin-amd64.tar.gz.sha256" ${INPUTDIR}
  checkIfFileExists "verrazzano-${VZ_DEV_VERSION}-linux-amd64.tar.gz" ${INPUTDIR}
  checkIfFileExists "verrazzano-${VZ_DEV_VERSION}-linux-amd64.tar.gz.sha256" ${INPUTDIR}
}

function verify_top_level_flavor_contents() {
    INPUT_DIR=$1
    checkIfFileExists "LICENSE" ${INPUT_DIR}
    checkIfFileExists "README.md" ${INPUT_DIR}
}

function verify_lite_bin() {
  INPUTDIR=$1
  checkIfFileExists "bom_utils.sh" $INPUTDIR/bin
  checkIfFileExists "vz" $INPUTDIR/bin
  checkIfFileExists "vz-registry-image-helper.sh" $INPUTDIR/bin
}

function validation_cleanup() {
  INPUTDIR=$1
  echo "Deleting directory: ${INPUTDIR}"
  rm -rf ${INPUTDIR}
}

if [ $TYPE == "Lite" ]
  then
    verify_lite_contents ${ROOTDIR}

    verify_top_level_flavor_contents ${LINUX_EXTRACTED}/${VZ_DIR}
    verify_lite_bin ${LINUX_EXTRACTED}/${VZ_DIR}
    verify_common_manifests ${LINUX_EXTRACTED}/${VZ_DIR}
    validation_cleanup $LINUX_EXTRACTED

    verify_top_level_flavor_contents ${DARWIN_EXTRACTED}/${VZ_DIR}
    verify_lite_bin ${DARWIN_EXTRACTED}/${VZ_DIR}
    verify_common_manifests ${DARWIN_EXTRACTED}/${VZ_DIR}
    validation_cleanup $DARWIN_EXTRACTED
  else
    verify_full_contents ${ROOTDIR}/${VZ_DIR}
    verify_full_bin ${ROOTDIR}/${VZ_DIR}
    verify_full_images ${ROOTDIR}/${VZ_DIR}
    verify_common_manifests ${ROOTDIR}/${VZ_DIR}
fi

# TODO verify BOM content for open source