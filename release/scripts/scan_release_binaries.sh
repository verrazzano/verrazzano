#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Perform malware scan on release binaries

set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

usage() {
    cat <<EOM
  Performs malware scan on release binaries.

  Usage:
    $(basename $0) <release branch> <directory where the release artifacts need to be downloaded> <flat to indicate whether to scan already downloaded bundle> <directory to place the scan report, which is optional>

  Example:
    $(basename $0) release-1.0 release_bundle_dir true

  The script expects the OCI CLI is installed. It also expects the following environment variables -
    RELEASE_VERSION - release version (major.minor.patch format, e.g. 1.0.1)
    OCI_REGION - OCI region
    OBJECT_STORAGE_NS - top-level namespace used for the request
    OBJECT_STORAGE_BUCKET - object storage bucket where the artifacts are stored
    SCANNER_ARCHIVE_LOCATION - command line scanner
    SCANNER_ARCHIVE_FILE - scanner archive
    VIRUS_DEFINITION_LOCATION - virus definition location
    NO_PROXY_SUFFIX - suffix for no_proxy environment variable
EOM
    exit 0
}

[ -z "$OCI_REGION" ] || [ -z "$OBJECT_STORAGE_NS" ] || [ -z "$OBJECT_STORAGE_BUCKET" ] ||
[ -z "$SCANNER_ARCHIVE_LOCATION" ] || [ -z "$SCANNER_ARCHIVE_FILE" ] || [ -z "$NO_PROXY_SUFFIX" ] ||
[ -z "$VIRUS_DEFINITION_LOCATION" ] || [ -z "$RELEASE_VERSION" ] || [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ "$1" == "-h" ] && { usage; }

. $SCRIPT_DIR/common.sh

if [ -z "$1" ]; then
  echo "Branch from where to download the release bundle, must be specified"
  exit 1
fi
BRANCH="$1"

if [ -z "$2" ]; then
  echo "Work directory must be specified"
  exit 1
fi
WORK_DIR="$2"

if [ -z "$3" ]; then
  echo "A flag to indicate whether to scan the release bundle which is already downloaded and extracted"
  exit 1
fi
USE_DOWNLOADED_BUNDLE="$3"

DEFAULT_SCAN_REPORT_DIR="$WORK_DIR/scan_report_dir"
SCAN_REPORT_DIR=${4:-$DEFAULT_SCAN_REPORT_DIR}

SCANNER_HOME="$WORK_DIR/scanner_home"
SCAN_REPORT="$SCAN_REPORT_DIR/scan_report.out"
SCAN_SUMMARY_REPORT="${SCAN_REPORT_DIR}/scan_summary.out"

VERRAZZANO_PREFIX="verrazzano-$RELEASE_VERSION"

RELEASE_TAR_BALL="$VERRAZZANO_PREFIX-lite.zip"
RELEASE_BUNDLE_DIR="$WORK_DIR/release_bundle"
DIR_TO_SCAN="$RELEASE_BUNDLE_DIR"

# Option to scan full bundle
if [ "${BUNDLE_TO_SCAN}" == "Full" ];then
  RELEASE_TAR_BALL="$VERRAZZANO_PREFIX.zip"
  DIR_TO_SCAN="$RELEASE_BUNDLE_DIR/$VERRAZZANO_PREFIX"
fi

function download_release_tarball() {
  cd $WORK_DIR
  mkdir -p $RELEASE_BUNDLE_DIR
  oci --region ${OCI_REGION} os object get \
        --namespace ${OBJECT_STORAGE_NS} \
        -bn ${OBJECT_STORAGE_BUCKET} \
        --name "${BRANCH}/${RELEASE_TAR_BALL}" \
        --file "$RELEASE_BUNDLE_DIR/${RELEASE_TAR_BALL}"
  SHA256_CMD="sha256sum -c"
  if [ "$(uname)" == "Darwin" ]; then
    SHA256_CMD="shasum -a 256 -c"
  fi
  ${SHA_CMD} ${_file}.256
  unzip ${_file}
  rm -f ${_file}
  rm -f ${_file}.256
  unzip $RELEASE_TAR_BALL
  rm $RELEASE_TAR_BALL
}

function install_scanner() {
  no_proxy="$no_proxy,${NO_PROXY_SUFFIX}"
  cd $SCANNER_HOME
  curl -O $SCANNER_ARCHIVE_LOCATION/$SCANNER_ARCHIVE_FILE
  tar --overwrite -xvf $SCANNER_ARCHIVE_FILE
}

function update_virus_definition() {
  VIRUS_DEF_FILE=$(curl -s $VIRUS_DEFINITION_LOCATION | grep -oP 'avvdat-.*?zip' | sort -nr | head -1)
  cd $SCANNER_HOME
  curl -O $VIRUS_DEFINITION_LOCATION/$VIRUS_DEF_FILE
  unzip -o $VIRUS_DEF_FILE
}

function scan_release_binaries() {
  mkdir -p $SCAN_REPORT_DIR
  if [ -e "$SCAN_REPORT" ]; then
    rm -f $SCAN_REPORT
  fi

  cd $DIR_TO_SCAN
  count_files=$(ls -1q *.* | wc -l)
  ls

  cd $SCANNER_HOME
  # The scan takes more than 50 minutes, the option --SUMMARY prints each and every file from all the layers, which is removed.
  # Also --REPORT option prints the output of the scan in the console, which is removed and redirected to a file
  echo "Starting the scan of $DIR_TO_SCAN, it might take a longer duration. The output of the scan is being written to $SCAN_REPORT ..."
  ./uvscan $DIR_TO_SCAN --RPTALL --RECURSIVE --CLEAN --UNZIP --VERBOSE --SUB --SUMMARY --PROGRAM --RPTOBJECTS >> $SCAN_REPORT 2>&1

  # Extract only the last 25 lines from the scan report and create a file, which will be used for the validation
  if [ -e "${SCAN_SUMMARY_REPORT}" ]; then
    rm -f $SCAN_SUMMARY_REPORT
  fi
  tail -25 ${SCAN_REPORT} > ${SCAN_SUMMARY_REPORT}

  # The following set of lines from the summary in the scan report is used here for validation.
  declare -a expectedLines=("Total files:...................     $count_files"
                            "Clean:.........................     $count_files"
                            "Not Scanned:...................     0"
                            "Possibly Infected:.............     0"
                            "Objects Possibly Infected:.....     0"
                            "Cleaned:.......................     0"
                            "Deleted:.......................     0")

  array_count=${#expectedLines[@]}
  echo "Count of expected lines: ${array_count}"
  result_count=0

  # Read the file scan_summary.out line by line and increment the counter when the line matches one of the expected lines defined above.
  while IFS= read -r line
  do
    for i in "${expectedLines[@]}"
    do
      case $line in
        *${i}*)
          result_count=$(($result_count+1))
          ;;
        *)
      esac
    done
  done < "$SCAN_SUMMARY_REPORT"
  echo "Count of expected lines in the scan summary: ${result_count}"
  if [ "$result_count" == "$array_count" ];then
    echo "Found all the expected lines in the summary of the scan report."
    return 0
  else
    echo "One or more expected lines are not found in the summary of the scan report, please check the complete report $SCAN_REPORT"
    return 1
  fi
}

mkdir -p $SCANNER_HOME
validate_oci_cli || exit 1

if [[ "${USE_DOWNLOADED_BUNDLE}" == "false" ]]; then
  download_release_tarball || exit 1
fi

install_scanner || exit 1
update_virus_definition || exit 1
scan_release_binaries || exit 1