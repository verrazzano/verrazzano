#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh
. $SCRIPT_DIR/logging.sh

function dump_header() {
  log "================================  DIAGNOSTIC OUTPUT START ================================="
  log ""
}

function dump_footer() {
  log ""
  log "================================  DIAGNOSTIC OUTPUT END ==================================="
}

# Dump specified objects based on described requirements
# $1 object type - i.e. namespaces, pods, jobs
# $2 namespace - namespace of the objects
# $3 object name regex - regex to retrieve certain jobs by name
# $4 (optional) fields - field selectors for kubectl organized as shown here: https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
# Usage:
# dump_objects "objectType" "namespace" "objectRegex" "fields"
function dump_objects() {
  local type=$1
  local namespace=$2
  local regex=$3
  local fields=$4

  if [[ -z "$type"  || -z "$namespace" ]]
  then
    error "Object type and namespace must be specified to describe objects."
    exit
  fi

  local object_names=($(kubectl get "${type}" --no-headers -o custom-columns=":metadata.name" --field-selector="${fields}" -n "${namespace}"| grep -E "${regex}"))

  dump_header

  for object in "${object_names[@]}"
  do
    log ""
    log "========================================================"
    log "Describing type: ${type}, name: ${object}"
    log "========================================================"
    kubectl describe "${type}" "${object}" -n "${namespace}"
  done

  dump_footer
}

# format the field selectors for a given array
# $1 selector - kubernetes selector: metadata.name, metadata.namespace, status.phase
# $2 eq - "=" or "!="
# $3 state - state of the object
# Usage:
# format_field_selectors "selector" "=" "status"
function format_field_selectors() {
  states=()
  for state in "${@:3}"
  do
    formatted_state="$(tr '[:lower:]' '[:upper:]' <<< ${state:0:1})$(tr '[:upper:]' '[:lower:]' <<< ${state:1})"
    states+=("${1}${2}${formatted_state}")
  done

  echo $(join , "${states[@]}")
}


# join an array with a specified value
# $1 join - value to join by
# $2 values - values in which to join
# Usage:
# join_by , "${ARRAY[@]}"
function join() {
  local IFS="$1"
  shift
  echo "$*"
}

# prints usage message for this script to consoleerr
# Usage:
# usage
function usage {
    error
    error "usage: $0 -o object_type -n namespace [-r name_regex] [-s state] [-S not_state] [-h]"
    error " -o object_type   Type of the object (i.e. namespaces, pods, jobs, etc)"
    error " -n namespace     Namespace of the given object type"
    error " -r name_regex    Regex to retrieve certain jobs by name"
    error " -s state         Specified state the described object should be in (i.e. Running)"
    error " -S not_state     Specified state that the described object should not be in"
    error " -h               Help"
    error
    exit 1
}

NAMESPACE="default"
NAME_REGEX=""
STATES=()
NOT_STATES=()
while getopts o:n:r:s:S:h flag
do
    case "${flag}" in
        o) OBJECT_TYPE=${OPTARG};;
        n) NAMESPACE=${OPTARG};;
        r) NAME_REGEX=${OPTARG};;
        s) STATES+=("${OPTARG}");;
        S) NOT_STATES+=("${OPTARG}");;
        h) usage;;
        *) usage;;
    esac
done
shift $((OPTIND -1))

STATE_FORMAT=$(format_field_selectors "status.phase" "=" "${STATES[@]}")
NOT_STATE_FORMAT=$(format_field_selectors "status.phase" "!=" "${NOT_STATES[@]}")
FIELD_SELECTORS=$(join , "$STATE_FORMAT" "$NOT_STATE_FORMAT")

dump_objects "${OBJECT_TYPE}" "${NAMESPACE}" "${NAME_REGEX}" "${FIELD_SELECTORS}"