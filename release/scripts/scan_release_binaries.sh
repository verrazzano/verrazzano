#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Perform malware scan on release binaries

set -e

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

usage() {
    cat <<EOM
  Performs malware scan on release binaries.

  Usage:
    $(basename $0) <release branch> <directory where the release artifacts need to be downloaded, defaults to the current directory>

  Example:
    $(basename $0) release-1.0

  The script expects the OCI CLI is installed. It also expects the following environment variables -
    OCI_REGION - OCI region
    OBJECT_STORAGE_NS - top-level namespace used for the request
    OBJECT_STORAGE_BUCKET - object storage bucket where the artifacts are stored
    SCANNER_ARCHIVE_LOCATION - McAfee command line scanner
    SCANNER_ARCHIVE_FILE - scanner archive
    VIRUS_DEF_LOCATION - virus definition location
    NO_PROXY_SUFFIX - suffix for no_proxy environment variable
EOM
    exit 0
}

[ -z "$OCI_REGION" ] || [ -z "$OBJECT_STORAGE_NS" ] || [ -z "$OBJECT_STORAGE_BUCKET" ] ||
[ -z "$SCANNER_ARCHIVE_LOCATION" ] || [ -z "$SCANNER_ARCHIVE_FILE" ] || [ -z "$NO_PROXY_SUFFIX" ] ||
[ -z "$VIRUS_DEF_LOCATION" ] || [ -z "$1" ] || [ -z "$2" ] || [ "$1" == "-h" ] && { usage; }

BRANCH=${1}
WORK_DIR=${2:-$SCRIPT_DIR}
SCAN_REPORT_DIR="$WORK_DIR/scan_report_dir"
SCANNER_HOME="$WORK_DIR/scanner_home"
SCAN_REPORT="$SCAN_REPORT_DIR/scan_report.out"
RELEASE_TAR_BALL="tarball.tar.gz"

function downaload_release_tarball() {
  cd $WORK_DIR
  oci --region ${OCI_REGION} os object get \
        --namespace ${OBJECT_STORAGE_NS} \
        -bn ${OBJECT_STORAGE_BUCKET} \
        --name "${BRANCH}/${RELEASE_TAR_BALL}" \
        --file "${RELEASE_TAR_BALL}"
}

function install_scanner() {
  no_proxy="$no_proxy,${NO_PROXY_SUFFIX}"
  cd $SCANNER_HOME
  curl -O $SCANNER_ARCHIVE_LOCATION/$SCANNER_ARCHIVE_FILE
  tar -xvf $SCANNER_ARCHIVE_FILE
}

function update_virus_definition() {
  VIRUS_DEF_FILE=$(curl -s $VIRUS_DEF_LOCATION | grep -oP 'avvdat-.*?zip' | sort -nr | head -1)
  cd $SCANNER_HOME
  curl -O $VIRUS_DEF_LOCATION/$VIRUS_DEF_FILE
  unzip $VIRUS_DEF_FILE
}

function scan_release_binaries() {
  mkdir -p $SCAN_REPORT_DIR
  cd $SCANNER_HOME
  # The scan takes more than 50 minutes
  ./uvscan $WORK_DIR/$RELEASE_TAR_BALL --RPTALL --RECURSIVE --CLEAN --UNZIP --VERBOSE --SUB --SUMMARY --PROGRAM --RPTOBJECTS --REPORT=$SCAN_REPORT

  # Extract only the last 50 lines from the scan report and create a file, which will be used for the validation
  local scan_summary="${SCAN_REPORT_DIR}/summary.log"
  tail -50 ${SCAN_REPORT} > ${scan_summary}

  # The following set of lines from the summary in the scan report is used here for validation.
  declare -a expectedLines=("Total files:...................     1"
                            "Clean:.........................     1"
                            "Not Scanned:...................     0"
                            "Possibly Infected:.............     0"
                            "Objects Possibly Infected:.....     0"
                            "Cleaned:.......................     0"
                            "Deleted:.......................     0")

  array_count=${#expectedLines[@]}
  echo "Count of expected lines: ${array_count}"
  result_count=0

  # Read the file scan_summary.log line by line and increment the counter when the line matches one of the expected lines defined above.
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
downaload_release_tarball || exit 1
install_scanner || exit 1
update_virus_definition || exit 1
scan_release_binaries || exit 1