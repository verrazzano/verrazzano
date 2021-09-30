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

[ -z "$OCI_REGION" ] || [ -z "$OCIR_REPOSITORY_BASE" ] || [ -z "$OCIR_COMPARTMENT_ID" ] || [ -z "$OCIR_PATH_FILTER" ] || -z "$SCAN_RESULTS_DIR" || [ "$1" == "-h" ] && { usage; }

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

# $1 Scan result OCID
# $2 Result file name
function get_scan_details() {
  echo $1
  echo $2
  oci vulnerability-scanning container scan result get --container-scan-result-id $1 --region $OCI_REGION > $2-details.json
  cat $2-details.json | jq -r '.data.problems[] | { sev: .severity, cve: ."cve-reference", description: .description } ' | jq -r '[.[]] | @csv' | sort -u > $2-details.csv
}

function get_all_scan_details() {
  get_repository_list
  get_scan_summaries
  check_for_missing_scans

  # TBD: If we iterate across the repository list here instead of the scan summary list, we can identify
  # missing scans in one pass, not doing that for now, keeping that check separate for now. But calling out
  # that we could do that if we always do them all at once

  # For each scan result in the scan summary list, fetch the full details
  while read CSV_LINE; do
    SCAN_RESULT_OCID=$(echo "$CSV_LINE" | cut -d, -f"5" | sed 's/"//g')
    RESULT_REPOSITORY_IMAGE=$(echo "$CSV_LINE" | cut -d, -f"2" | sed 's/"//g' | sed 's;/;_;g')
    RESULT_IMAGE=$(echo "$CSV_LINE" | cut -d, -f"2" | sed 's/"//g')
    # TBD: Not great but should ensure unique files as a start here
    RESULT_FILE_PREFIX=$(echo "$SCAN_RESULTS_DIR/$SCAN_RESULT_OCID")
    get_scan_details $SCAN_RESULT_OCID $RESULT_FILE_PREFIX
  done <$SCAN_RESULTS_DIR/scan-all-summary.csv
}

# Validate OCI CLI
validate_oci_cli || exit 1

mkdir -p $SCAN_RESULTS_DIR

get_all_scan_details || exit 1
