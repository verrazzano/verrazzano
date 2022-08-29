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
    $(basename $0) <release branch> <directory where the release artifacts need to be downloaded, defaults to the current directory> <release version>

  Example:
    $(basename $0) release-1.0 . 1.0.2

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

BRANCH=${1}
WORK_DIR=${2:-$SCRIPT_DIR}

SCAN_REPORT_DIR="$WORK_DIR/scan_report_dir"
SCANNER_HOME="$WORK_DIR/scanner_home"
SCAN_REPORT="$SCAN_REPORT_DIR/scan_report.out"
RELEASE_TAR_BALL="verrazzano-$RELEASE_VERSION-open-source.zip"

# Option to scan commercial bundle
if [ "${BUNDLE_TO_SCAN}" == "commercial" ];then
  RELEASE_TAR_BALL="verrazzano-$RELEASE_VERSION-commercial.zip"
fi

RELEASE_BUNDLE_DIR="$WORK_DIR/release_bundle"

function download_release_tarball() {
  cd $WORK_DIR
  mkdir -p $RELEASE_BUNDLE_DIR
  echo "Downloading release bundle to $RELEASE_BUNDLE_DIR/${RELEASE_TAR_BALL} ..."
  oci --region ${OCI_REGION} os object get \
        --namespace ${OBJECT_STORAGE_NS} \
        -bn ${OBJECT_STORAGE_BUCKET} \
        --name "${BRANCH}/${RELEASE_TAR_BALL}" \
        --file "$RELEASE_BUNDLE_DIR/${RELEASE_TAR_BALL}"
  echo "Successfully downloaded the release bundle"
  ls -ltr $RELEASE_BUNDLE_DIR
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

  # Extract the release bundle to a directory, and scan that directory
  cd $RELEASE_BUNDLE_DIR
  unzip $RELEASE_TAR_BALL
  rm $RELEASE_TAR_BALL

  count_files=$(ls -1q *.* | wc -l)
  ls $RELEASE_BUNDLE_DIR

  cd $SCANNER_HOME
  # The scan takes more than 50 minutes, the option --SUMMARY prints each and every file from all the layers, which is removed.
  # Also --REPORT option prints the output of the scan in the console, which is removed and redirected to a file
  echo "Starting the scan of $RELEASE_BUNDLE_DIR, it might take a longer duration. The output of the scan is being written to $SCAN_REPORT ..."
  ./uvscan $RELEASE_BUNDLE_DIR --RPTALL --RECURSIVE --CLEAN --UNZIP --VERBOSE --SUB --SUMMARY --PROGRAM --RPTOBJECTS >> $SCAN_REPORT 2>&1

  # Extract only the last 25 lines from the scan report and create a file, which will be used for the validation
  local scan_summary="${SCAN_REPORT_DIR}/scan_summary.out"
  if [ -e "${scan_summary}" ]; then
    rm -f $scan_summary
  fi
  tail -25 ${SCAN_REPORT} > ${scan_summary}

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
  done < "$scan_summary"
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
download_release_tarball || exit 1
#install_scanner || exit 1
#update_virus_definition || exit 1
#scan_release_binaries || exit 1