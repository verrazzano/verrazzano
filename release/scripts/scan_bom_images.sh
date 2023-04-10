#!/usr/bin/env bash
#
# Copyright (c) 2021, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Scan images in a specified BOM

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
TOOL_SCRIPT_DIR=${SCRIPT_DIR}/../../tools/scripts

. $SCRIPT_DIR/common.sh

ALLOW_LIST=
BOM_FILE=
REGISTRY=
REPO_PATH=
OUTPUT_DIR=
OCIR_NAMESPACE=

# The COMBINED_PATH is formed based on whether there is an OCIR tenancy namespace or not:
#       $REGISTRY/$REPO_PATH
#       $REGISTRY/$OCIR_NAMESPACE/$REPO_PATH
COMBINED_PATH=

function usage() {
  ec=${1:-0}
  if [ ! -z "$2" ]; then
    echo "$2"
  fi

  cat << EOM
  Scan images in specified BOM

  Usage:
    $(basename $0) -b bom-file -r results-directory [-a allow-list]

  Options:
    -b <bom-file>           BOM file to use for determining images to scan (required)
    -o <output-directory>   Directory to store scan result files (required)
    -r <docker-registry>    Docker registry to find images in (required)
    -n <ocir-namespace>     OCIR namespace. This is required if the registry is an OCIR registry, and is not needed otherwise (optional)
    -p <repository-path>    Repository name/prefix for each image, e.g \"path/to/my/image\"; not including an OCIR namespace if one is required, if not specified the default will be used according to the BOM
    -x <ocir-repo-path>     This is the OCIR namespace plus the repository path as well (basically -n and -p combined).
    -a <allow-list>         Allow-list to use for scanning (optional)

    One of the path options must be specified: -x or -p

  Pre-requisites:
    This requires that trivy and grype scanners are installed

  Example:
    $(basename $0) -b verrazzano-bom.json -t ghcr.io -p my/repo/path -o scan-results
    $(basename $0) -b verrazzano-bom.json -t phx.ocir.io -n tenancynamespace -p my/repo/path -o scan-results
EOM
  exit ${ec}
}

# This will get the scan summaries and details for all of the repositories
#
# It will also verify that all repositories found have scan results as well
#
# $1 Scan BOM file
function scan_images_in_bom() {
  # Get a list of the images in the BOM based on the specified registry and repo path, but trim the paths
  # off in the file so we are left with them in the form image:tag
  local bomimages=$(mktemp temp-bom-images-XXXXXX.out)
  sh $TOOL_SCRIPT_DIR/vz-registry-image-helper.sh -m $bomimages -t $REGISTRY -r $COMBINED_PATH -b $BOM_FILE
  cat $bomimages
  echo "$REGISTRY/$COMBINED_PATH"
  sed -i "s;$REGISTRY/$COMBINED_PATH/;;g" $bomimages

  cat $bomimages

  # Scan each image listed in the BOM
  while read BOM_IMAGE; do
    RESULT_REPOSITORY_IMAGE=$(echo "${BOM_IMAGE}" | sed 's;/;_;g')
    RESULT_FILE_PREFIX=$(echo "${OUTPUT_DIR}/${RESULT_REPOSITORY_IMAGE}")
    SCAN_IMAGE=$(echo "${REGISTRY}/${COMBINED_PATH}/${BOM_IMAGE}")

    # FIXME: Add allowlist support

    echo "performing trivy scan for $SCAN_IMAGE"
    trivy image -f json -o ${RESULT_FILE_PREFIX}-trivy-details.json ${SCAN_IMAGE} 2> ${RESULT_FILE_PREFIX}-trivy.err || echo "trivy scan failed for $SCAN_IMAGE"
    cat ${RESULT_FILE_PREFIX}-trivy-details.json | jq -r '.Results[].Vulnerabilities[] | { sev: .Severity, cve: ."VulnerabilityID", description: .Description } ' | sed 's/\\[nt]/ /g' | jq -r '[.[]] | @csv' | sort -u > ${RESULT_FILE_PREFIX}-trivy-details.csv

    echo "performing grype scan for $BOM_IMAGE"
    grype ${SCAN_IMAGE} -o json > ${RESULT_FILE_PREFIX}-grype-details.json 2> ${RESULT_FILE_PREFIX}-grype.err || echo "grype scan failed for $SCAN_IMAGE"
    cat ${RESULT_FILE_PREFIX}-grype-details.json | jq -r '.matches[] | { sev: .vulnerability.severity, cve: .vulnerability.id, description: .vulnerability.description } ' | jq -r '[.[]] | @csv' | sort -u > ${RESULT_FILE_PREFIX}-grype-details.csv
  done <$bomimages
  rm $bomimages
}

function validate_inputs() {
  if [ -z "$BOM_FILE" ] || [ ! -f "$BOM_FILE" ]; then
    usage 1 "A valid BOM file is required to be specified"
  fi

  if [ -z "$OUTPUT_DIR" ]; then
    usage 1 "A valid scan results directory is required to be specified"
  else
    if [ ! -d "$OUTPUT_DIR" ]; then
      mkdir -p $OUTPUT_DIR || usage 1 "Failed creating $OUTPUT_DIR. A valid scan results directory location is required to be specified"
    fi
  fi

  if [ -z "$REGISTRY" ]; then
    usage 1 "The registry must be specified"
  fi

  if [ -z "$REPO_PATH" ]; then
    usage 1 "The repository path must be specified"
  fi

  # If we already have a combined path, we were give -x with a namespace and path already
  # If not, then we check if they gave us -p or -n and -p
  if [ -z "$COMBINED_PATH" ]; then
    if [ -z "$OCIR_NAMESPACE" ]; then
      COMBINED_PATH="$REPO_PATH"
    else
      COMBINED_PATH="$OCIR_NAMESPACE/$REPO_PATH"
    fi
  fi
}

function verify_prerequisites() {
  command -v trivy >/dev/null 2>&1 || {
    usage 1 "trivy scanner is not installed"
  }
  command -v grype >/dev/null 2>&1 || {
    usage 1 "grype scanner is not installed"
  }
  echo "Grype installation directory: $(which grype)"
  echo "Trivy installation directory: $(which trivy)"
}

while getopts 'hb:a:o:n:r:p:x:' opt; do
  case $opt in
  a)
    ALLOW_LIST=$OPTARG
    ;;
  b)
    BOM_FILE=$OPTARG
    ;;
  n)
    OCIR_NAMESPACE=$OPTARG
    ;;
  o)
    OUTPUT_DIR=$OPTARG
    ;;
  r)
    REGISTRY=$OPTARG
    ;;
  p)
    REPO_PATH=$OPTARG
    ;;
  x)
    COMBINED_PATH=$OPTARG
    REPO_PATH=$(echo $COMBINED_PATH | cut -d "/" -f2-)
    ;;
  h | ?)
    usage
    ;;
  esac
done

validate_inputs
verify_prerequisites
scan_images_in_bom || exit 1
