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
    $(basename $0) <directory containing the release artifacts> <directory to download the scanner> <directory to place the scan report>

  Example:
    $(basename $0) release_bundle_dir scanner_home scan_report_dir

  The script expects the following environment variables -
    RELEASE_VERSION - release version (major.minor.patch format, e.g. 1.0.1)
    SCANNER_ARCHIVE_LOCATION - command line scanner
    SCANNER_ARCHIVE_FILE - scanner archive
    VIRUS_DEFINITION_LOCATION - virus definition location
    NO_PROXY_SUFFIX - suffix for no_proxy environment variable
EOM
    exit 0
}

[ -z "$SCANNER_ARCHIVE_LOCATION" ] || [ -z "$SCANNER_ARCHIVE_FILE" ] || [ -z "$NO_PROXY_SUFFIX" ] ||
[ -z "$VIRUS_DEFINITION_LOCATION" ] || [ -z "$RELEASE_VERSION" ] || [ "$1" == "-h" ] && { usage; }

. $SCRIPT_DIR/common.sh

if [ -z "$1" ]; then
  echo "Directory to download the bundle or directory containing the release bundle extracted"
  exit 1
fi
RELEASE_BUNDLE_DOWNLOAD_DIR="$1"

if [ -z "$2" ]; then
  echo "Directory to place the scaner"
  exit 1
fi
SCANNER_HOME="$2"

if [ -z "$3" ]; then
  echo "Directory to place the scan report"
  exit 1
fi
SCAN_REPORT_DIR="$3"

DIR_TO_SCAN="$RELEASE_BUNDLE_DOWNLOAD_DIR"

# When an environment variable BUNDLE_TO_SCAN is set to Full, the script scans the full bundle
# The variable DIR_TO_SCAN is redefined as there will be a top level verrazzano-<major>.<minor>.<patch> directory inside the full bundle
if [ "${BUNDLE_TO_SCAN}" == "Full" ];then
  VERRAZZANO_PREFIX="verrazzano-$RELEASE_VERSION"
  DIR_TO_SCAN="$RELEASE_BUNDLE_DOWNLOAD_DIR/$VERRAZZANO_PREFIX"
fi

SCAN_REPORT="$SCAN_REPORT_DIR/scan_report.out"

function install_scanner() {
  mkdir -p $SCANNER_HOME
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
  ls
  count_files=$(ls -1q *.* | wc -l)

  cd $SCANNER_HOME
  # The scan takes more than 50 minutes, the option --SUMMARY prints each and every file from all the layers, which is removed.
  # Also --REPORT option prints the output of the scan in the console, which is removed and redirected to a file
  echo "Starting the scan of $DIR_TO_SCAN, it might take a longer duration."
  echo "The output of the scan is being written to $SCAN_REPORT ..."
  ./uvscan $DIR_TO_SCAN --RPTALL --RECURSIVE --CLEAN --UNZIP --VERBOSE --SUB --SUMMARY --PROGRAM --RPTOBJECTS >> $SCAN_REPORT 2>&1

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

install_scanner || exit 1
update_virus_definition || exit 1
scan_release_binaries || exit 1
