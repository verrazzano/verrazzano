#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Generate OCIR image scan report

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

usage() {
    cat <<EOM
  Generates OCIR image scan report

  Usage:
    $(basename $0) FIXME

  Example:
    $(basename $0) FIXME

  The script expects the OCI CLI is installed. It also expects the following environment variables -
    OCI_REGION - OCI region
    OCIR_REPOSITORY_BASE - Base OCIR repository path
    OCIR_COMPARTMENT_ID - Compartment the OCIR repository is in
    OCIR_PATH_FILTER - Regular expression to limit repository paths to include in report
    SCAN_RESULTS_DIR
EOM
    exit 0
}

[ -z "$OCI_REGION" ] || [ -z "$OCIR_REPOSITORY_BASE" ] || [ -z "$OCIR_COMPARTMENT_ID" ] || [ -z "$OCIR_PATH_FILTER" ] || [ -z "$SCAN_RESULTS_DIR" ] || [ "$1" == "-h" ] && { usage; }

function get_repository_list() {
  # TBD: See if we can just filter of the OCI list results to use the path filter, limit the json as well
  oci artifacts container repository list --compartment-id $OCIR_COMPARTMENT_ID --region $OCI_REGION --all > $SCAN_RESULTS_DIR/scan-all-repos.json
  cat $SCAN_RESULTS_DIR/scan-all-repos.json | jq '.data.items[]."display-name" | select(test("$OCIR_PATH_FILTER")?)' > $SCAN_RESULTS_DIR/filtered-repository-list.out
}

function get_scan_summaries() {
  # TBD: Add filtering here
  # TBD: Need to add more fields here so we can at least have the result OCIDs and may also want times in case there are multiple scan results to differentiate
  # TBD: For multiple scans assuming -u will be mostly a noop here, ie: if we include all fields we wouldn't see any duplicates
  oci vulnerability-scanning container scan result list --compartment-id $OCIR_COMPARTMENT_ID --region $OCI_REGION --all > $SCAN_RESULTS_DIR/scan-all-summary.json
  cat $SCAN_RESULTS_DIR/scan-all-summary.json | jq -r '.data.items[] | { sev: ."highest-problem-severity", repo: .repository, image: .image, count: ."problem-count", id: .id } ' | jq -r '[.[]] | @csv' | sort -u > $SCAN_RESULTS_DIR/scan-all-summary.csv
}

function check_for_missing_scans() {
  # TBD: Add check here, basically we iterate through the repository list and ensure that we have scan results
  #      for the repository in the summary. If we don't flag each missing one
  echo "TBD"
}

# This will generate a more human readable text report. More suitable for forming a BUG report with than the CSV alone.
#
# $1 Scan result OCID
# $2 Result file name
# $1 Scan result severity
# $2 Repository image
# $3 Image tag
# $4 Issue count
# $5 Scan result OCID
# $6 Result file basename (path and file prefix to use)
function generate_detail_text_report() {
  [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ] || [ -z "$5" ] || [ -z "$6" ] || [ -z "$7" ] && { echo "ERROR: generate_detail_text_report invalid args: $1 $2 $3 $4 $5 $6 $7"; return }
  RESULT_SEVERITY=$1
  RESULT_REPOSITORY_IMAGE=$2
  RESULT_IMAGE_TAG=$3
  RESULT_COUNT=$4
  SCAN_RESULT_OCID=$5
  RESULT_FILE_BASE=$6
  TIME_FINISHED=$7
  # REVIEW: Rudimentary for now, can work on the format later, etc...
  echo "OCIR Result Scan ID:  $SCAN_RESULT_OCID" > $RESULT_FILE_BASE-report.out
  echo "Scan Finished:        $TIME_FINISHED" > $RESULT_FILE_BASE-report.out
  echo "Image:                $RESULT_REPOSITORY_IMAGE:$RESULT_IMAGE_TAG" > $RESULT_FILE_BASE-report.out
  echo "Issue Count:          $RESULT_COUNT" > $RESULT_FILE_BASE-report.out
  echo "Highest Severity:     $RESULT_SERVERITY" > $RESULT_FILE_BASE-report.out
  echo "" > $RESULT_FILE_BASE-report.out
  echo "Issues:" > $RESULT_FILE_BASE-report.out
  cat $RESULT_FILE_BASE-details.csv > $RESULT_FILE_BASE-report.out
}

# This will get the detailed scan results in JSON, form a CSV report, and also form a more human readable report
#
# $1 Scan result severity
# $2 Repository image
# $3 Image tag
# $4 Issue count
# $5 Scan result OCID
# $6 Result file basename (path and file prefix to use)
function get_scan_details() {
  [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ] || [ -z "$5" ] || [ -z "$6" ] && { echo "ERROR: get_scan_details invalid args: $1 $2 $3 $4 $5 $6"; return }
  RESULT_SEVERITY=$1
  RESULT_REPOSITORY_IMAGE=$2
  RESULT_IMAGE_TAG=$3
  RESULT_COUNT=$4
  SCAN_RESULT_OCID=$5
  RESULT_FILE_BASE=$6
  oci vulnerability-scanning container scan result get --container-scan-result-id $5 --region $OCI_REGION > $RESULT_FILE_BASE-details.json
  cat $RESULT_FILE_BASE-details.json | jq -r '.data.problems[] | { sev: .severity, cve: ."cve-reference", description: .description } ' | jq -r '[.[]] | @csv' | sort -u > $RESULT_FILE_BASE-details.csv
  TIME_FINISHED=$(cat $RESULT_FILE_BASE-details.json | jr -r '.data."time-finished"')
  generate_detail_text_report $1 $2 $3 $4 $5 $6 $7
}

# This will get the scan summaries and details for all of the repositories
#
# It will also verify that all repositories found have scan results as well
function get_all_scan_details() {
  get_repository_list
  get_scan_summaries
  check_for_missing_scans

  # TBD: If we iterate across the repository list here instead of the scan summary list, we can identify
  # missing scans in one pass, not doing that for now, keeping that check separate for now. But calling out
  # that we could do that if we always do them all at once

  # For each scan result in the scan summary list, fetch the full details
  while read CSV_LINE; do
    RESULT_SEVERITY=$(echo "$CSV_LINE" | cut -d, -f"1" | sed 's/"//g')
    RESULT_REPOSITORY_IMAGE=$(echo "$CSV_LINE" | cut -d, -f"2" | sed 's/"//g' | sed 's;/;_;g')
    RESULT_IMAGE_TAG=$(echo "$CSV_LINE" | cut -d, -f"3" | sed 's/"//g')
    RESULT_COUNT=$(echo "$CSV_LINE" | cut -d, -f"4" | sed 's/"//g')
    SCAN_RESULT_OCID=$(echo "$CSV_LINE" | cut -d, -f"5" | sed 's/"//g')\

    # REVIEW: Not great but should ensure unique files as a start here (see if we can use image names reliably here instead, but
    #   we need to correlate the details and report to the exact scan results which are identified using an OCID)
    RESULT_FILE_PREFIX=$(echo "$SCAN_RESULTS_DIR/$SCAN_RESULT_OCID")
    get_scan_details $RESULT_SEVERITY $RESULT_REPOSITORY_IMAGE $RESULT_IMAGE_TAG $RESULT_COUNT $SCAN_RESULT_OCID $RESULT_FILE_PREFIX
  done <$SCAN_RESULTS_DIR/scan-all-summary.csv
}

# Validate OCI CLI
validate_oci_cli || exit 1

mkdir -p $SCAN_RESULTS_DIR

get_all_scan_details || exit 1
