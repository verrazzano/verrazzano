#!/usr/bin/env bash
#
# Copyright (c) 2021, 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Generate OCIR image scan report

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
TOOL_SCRIPT_DIR=${SCRIPT_DIR}/../../tools/scripts

. $SCRIPT_DIR/common.sh

usage() {
    cat <<EOM
  Generates OCIR image scan report

  Usage:
    $(basename $0) scan-bom-file

  Example:
    $(basename $0) verrazzano-bom.json

  The script expects the OCI CLI is installed. It also expects the following environment variables -
    OCI_REGION - OCI region
    OCIR_SCAN_REGISTRY - OCIR Registry
    OCIR_REPOSITORY_BASE - Base OCIR repository path
    OCIR_COMPARTMENT_ID - Compartment the OCIR repository is in
    OCIR_PATH_FILTER - Regular expression to limit repository paths to include in report
    SCAN_RESULTS_DIR
EOM
    exit 0
}

[ -z "$OCI_REGION" ] || [ -z "$OCIR_SCAN_REGISTRY" ] || [ -z "$OCIR_REPOSITORY_BASE" ] || [ -z "$OCIR_COMPARTMENT_ID" ] || [ -z "$OCIR_PATH_FILTER" ] \
|| [ -z "$SCAN_RESULTS_DIR" ] || [ "$1" == "-h" ] || [ ! -f "$1" ]&& { usage; }

function get_repository_list() {
  # TBD: See if we can just filter of the OCI list results to use the path filter, limit the json as well
  oci artifacts container repository list --compartment-id $OCIR_COMPARTMENT_ID --region $OCI_REGION --all > $SCAN_RESULTS_DIR/scan-all-repos.json
  cat $SCAN_RESULTS_DIR/scan-all-repos.json | jq '.data.items[]."display-name" | select(test("$OCIR_PATH_FILTER")?)' > $SCAN_RESULTS_DIR/filtered-repository-list.out
}

function get_scan_summaries() {
  # TBD: Add filtering here
  # TBD: Need to add more fields here so we can at least have the result OCIDs and may also want times in case there are multiple scan results to differentiate
  # TBD: For multiple scans assuming -u will be mostly a noop here, ie: if we include all fields we wouldn't see any duplicates
  oci vulnerability-scanning container scan result list --compartment-id $OCIR_COMPARTMENT_ID --region $OCI_REGION --all > $SCAN_RESULTS_DIR/ocir-scan-all-summary.json
  cat $SCAN_RESULTS_DIR/ocir-scan-all-summary.json | jq -r '.data.items[] | { finished: ."time-finished", sev: ."highest-problem-severity", full: (.repository + ":" + .image), repo: .repository, image: .image, count: ."problem-count", id: .id } ' | jq -r '[.[]] | @csv' | sort -u > $SCAN_RESULTS_DIR/ocir-scan-all-summary.csv
}

# This will generate a more human readable text report. More suitable for forming a BUG report with than the CSV alone.
#
# $1 Scan result severity
# $2 Repository image with tag
# $3 Issue count
# $4 Scan result OCID
# $5 Result file basename (path and file prefix to use)
# $6 time finished
# $7 Overall Summary Report File
function generate_detail_text_report() {
  [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ] || [ -z "$5" ] || [ -z "$6" ] || [ -z "$7" ] && { echo "ERROR: generate_detail_text_report invalid args: $1 $2 $3 $4 $5 $6 $7"; return; }
  RESULT_SEVERITY=$1
  RESULT_REPOSITORY_IMAGE=$2
  RESULT_COUNT=$3
  SCAN_RESULT_OCID=$4
  RESULT_FILE_BASE=$5
  TIME_FINISHED=$6
  OVERALL_SUMMARY=$7
  # REVIEW: Rudimentary for now, can work on the format later, etc...
  echo "OCIR Result Scan ID:  $SCAN_RESULT_OCID" > $RESULT_FILE_BASE-ocir-report.out
  echo "Scan Finished:        $TIME_FINISHED" >> $RESULT_FILE_BASE-ocir-report.out
  echo "Image:                $RESULT_REPOSITORY_IMAGE" >> $RESULT_FILE_BASE-ocir-report.out
  echo "Issue Count:          $RESULT_COUNT" >> $RESULT_FILE_BASE-ocir-report.out
  echo "Highest Severity:     $RESULT_SEVERITY" >> $RESULT_FILE_BASE-ocir-report.out
  echo "Issues:" >> $RESULT_FILE_BASE-ocir-report.out
  cat $RESULT_FILE_BASE-ocir-details.csv >> $RESULT_FILE_BASE-ocir-report.out

  # Contribute a subset of details to the overall summary report. This includes only CRITICAL and HIGH CVE's
  echo "+++++"
  echo "Image:                $RESULT_REPOSITORY_IMAGE" >> $OVERALL_SUMMARY
  cat $RESULT_FILE_BASE-ocir-details.csv | grep -e 'CRITICAL' -e 'HIGH' >> $OVERALL_SUMMARY
  echo "-----"
}

# This will get the detailed scan results in JSON, form a CSV report, and also form a more human readable report
#
# $1 Scan result severity
# $2 Repository image with tag
# $3 Issue count
# $4 Scan result OCID
# $5 Result file path
# $6 Overall Summary Report File
function get_scan_details() {
  [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ] || [ -z "$5" ] || [ -z "$6" ] && { echo "ERROR: get_scan_details invalid args: $1 $2 $3 $4 $5 $6"; return; }
  RESULT_SEVERITY=$1
  RESULT_REPOSITORY_IMAGE=$2
  RESULT_COUNT=$3
  SCAN_RESULT_OCID=$4
  RESULT_FILE_PREFIX="$5/$RESULT_REPOSITORY_IMAGE"
  oci vulnerability-scanning container scan result get --container-scan-result-id $4 --region $OCI_REGION > $RESULT_FILE_PREFIX-ocir-details.json
  cat $RESULT_FILE_PREFIX-ocir-details.json | jq -r '.data.problems[] | { sev: .severity, cve: ."cve-reference", description: .description } ' | sed 's/\\[nt]/ /g' | jq -r '[.[]] | @csv' | sort -u > $RESULT_FILE_PREFIX-ocir-details.csv
  TIME_FINISHED=$(cat $RESULT_FILE_PREFIX-ocir-details.json | jq -r '.data."time-finished"')
  generate_detail_text_report $1 $2 $3 $4 $RESULT_FILE_PREFIX $TIME_FINISHED $6
}

# This will get the scan summaries and details for all of the repositories
#
# It will also verify that all repositories found have scan results as well
#
# $1 Scan BOM file
function get_all_scan_details() {
  [ ! -f "$1" ] && { echo "ERROR: get_all_scan_details invalid args: $1"; return; }

  local bomimages=$(mktemp temp-bom-images-XXXXXX.out)
  local overallsummary=$SCAN_RESULTS_DIR/overall-scan-ocir-report.out
  sh $TOOL_SCRIPT_DIR/vz-registry-image-helper.sh -m $bomimages -t $OCIR_SCAN_REGISTRY -r $OCIR_REPOSITORY_BASE -b $1

  # trim off the registry and base info so the images we have can be used for lookups in the CSV data
  sed -i "s;$OCIR_SCAN_REGISTRY/$OCIR_REPOSITORY_BASE/;;g" $bomimages

  # Get the scan summaries
  get_scan_summaries

  # For each image listed in the BOM, find the summary entries in the CSV list
  while read BOM_IMAGE; do
    echo "Getting scan details for $BOM_IMAGE"

    # Find all scan summary entries for the image
    local imagecsv=(mktemp temp_image-csv-XXXXXX.csv)
    grep $BOM_IMAGE $SCAN_RESULTS_DIR/ocir-scan-all-summary.csv > $imagecsv

    if [ ! -s "$imagecsv" ]; then
      echo "ERROR: No scan results found for $BOM_IMAGE"
      echo "$BOM_IMAGE" >> $SCAN_RESULTS_DIR/IMAGES-MISSING-OCIR-SCANS.OUT
    else
      # The summary is sorted ascending with the finished time as the first field, so get the last non-empty line
      # of the CSV matches for this image for the most recent scan
      CSV_LINE=$(tail -n 1 $imagecsv)
      RESULT_FINISHED=$(echo "$CSV_LINE" | cut -d, -f"1" | sed 's/"//g')
      RESULT_SEVERITY=$(echo "$CSV_LINE" | cut -d, -f"2" | sed 's/"//g')
      RESULT_REPOSITORY_IMAGE=$(echo "$BOM_IMAGE" | sed 's;/;_;g')
      RESULT_COUNT=$(echo "$CSV_LINE" | cut -d, -f"6" | sed 's/"//g')
      SCAN_RESULT_OCID=$(echo "$CSV_LINE" | cut -d, -f"7" | sed 's/"//g')

      # We only are reporting the last scan for the specific tagged image, so we should be OK using the image name/tag here for the filename)
      get_scan_details $RESULT_SEVERITY $RESULT_REPOSITORY_IMAGE $RESULT_COUNT $SCAN_RESULT_OCID $SCAN_RESULTS_DIR $overallsummary
    fi
    rm $imagecsv
  done <$bomimages
  rm $bomimages
}

# Validate OCI CLI
validate_oci_cli || exit 1

mkdir -p $SCAN_RESULTS_DIR

get_all_scan_details $1 || exit 1
