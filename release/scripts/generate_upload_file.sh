#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

declare reportName="Daily Scan Results"
declare reportType="VZ_DAILY_SCAN"
declare reportRelease=
declare reportBranch=
declare reportTimestamp=
declare resultVersion=

get_timestamp_from_csv_file () {
    local csv_file="${1}"
    #head -1 ${csv_file} | cut -f4 -d,
    # temporarily override date for testing
    date -u +"%Y-%m-%dT%H:%M:%SZ"
    # don't try to reformat the date: -j flag isn't portable, and the csv file now has a UTC date in the right format
    # local date_string=$(head -1 ${csv_file} | cut -f4 -d,)
    # date -j -f "%Y%m%d%H%M%S" "${date_string}" +"%Y-%m-%dT%H:%M:%S%z" # date with tz offset
    # date -j -f "%Y%m%d%H%M%S" "${date_string}" +"%Y-%m-%dT%H:%M:%SZ"	# date hard-wired to Z (UTC time)
}

get_branch_from_csv_file () {
    local csv_file="${1}"
    head -1 ${csv_file} | cut -f2 -d,
}

get_release_from_branch_name () {
    local branch_name="${1}"
    local url_prefix="https://raw.githubusercontent.com/verrazzano/verrazzano"
    local version_file=".verrazzano-development-version"
    curl -s ${url_prefix}/${branch_name}/${version_file} | grep -- "^verrazzano-development-version=" | cut -f2 -d=
}

output_report_prologue () {
    echo "{"
    echo "    \"reportType\": \"${reportType}\","
    echo "    \"reportName\": \"${reportName}\","
    echo "    \"reportRelease\": \"${reportRelease}\","
    echo "    \"reportBranch\": \"${reportBranch}\","
    echo "    \"reportTimestamp\": \"${reportTimestamp}\","
    echo "    \"reportResults\": ["
}

output_report_epilogue () {
    echo "        }"
    echo "    ]"
    echo "}"
}

output_scan_result () {

    oIFS="$IFS"
    IFS=","
    set -- ${1}
    IFS="${oIFS}"

    if [[ ${_firstLine} = false ]] ; then
        echo "        },"
    fi

    echo "        {"
    echo "            \"vulnerabilityId\" : \"${7}\","
    echo "            \"vulnerabilitySeverity\" : \"${8}\","
    echo "            \"reportingScanner\" : \"${6}\","
    echo "            \"artifactName\" : \"${9%%:*}\","
    echo "            \"artifactVersion\" : \"${9#*:}\","
    echo "            \"verrazzanoVersion\" : \"${resultVersion}\""
}

declare _inputFile="${1}"
if [[ -z ${_inputFile} || ! -f ${_inputFile} ]] ; then
    echo "Input file '${_inputFile}' not found"
    exit 1
fi

if [[ -n "${2}" ]] ; then
    reportName="${reportName} (${2})"
fi

reportBranch=$(get_branch_from_csv_file ${_inputFile})
reportRelease=$(get_release_from_branch_name ${reportBranch})
reportTimestamp=$(get_timestamp_from_csv_file ${_inputFile})
resultVersion=$(echo ${reportRelease} | cut -f1,2 -d.)
reportRelease=${resultVersion}

output_report_prologue

declare _line=
declare _firstLine=first
cat ${_inputFile} | while read _line
do
    output_scan_result "${_line}"
    _firstLine=false
done

output_report_epilogue

