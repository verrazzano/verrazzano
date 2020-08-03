#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

export DIAGNOSTIC_LOG="${DIAGNOSTIC_LOG:-${SCRIPT_DIR}/build/logs/diagnostics.log}"
export LOG_FILE="${DIAGNOSTIC_LOG}"
. $SCRIPT_DIR/common.sh


# Dump Diagnostic header with message
# $1 message - message given by failure to identify cause
# Usage:
# dump_header "message"
function dump_header() {
  local message=$1

  if [ -z "$message" ] ; then
    log "================================  DIAGNOSTIC OUTPUT START ================================="
    log ""
  else
    log "================================  DIAGNOSTIC OUTPUT START: ${message} ================================="
    log ""
  fi
}

# Dump Diagnostic footer
# Usage:
# dump_footer
function dump_footer() {
  log ""
  log "================================  DIAGNOSTIC OUTPUT END ==================================="
}

# Dump specified objects based on described requirements
# $1 command - command type specified
# $2 object type - i.e. namespaces, pods, jobs
# $3 namespace - namespace of the objects
# $4 object name regex - regex to retrieve certain jobs by name
# $5 (optional) fields - field selectors for kubectl organized as shown here: https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
# $6 (optional) message - dump header message to inform the cause of the output
# $7 (optional) container - container in which the logs should be retrieved
# Usage:
# dump_objects "command" "objectType" "namespace" "objectRegex" "fields" "message" "container"
function dump_objects() {
  local command=$1
  local type=$2
  local namespace=$3
  local regex=$4
  local fields=$5
  local message=$6
  local container=$7

  if [[ -z "$type"  || -z "$namespace" ]] ; then
    error "Object type and namespace must be specified to describe objects."
    exit 1
  fi

  local object_names=($(kubectl get "${type}" --no-headers -o custom-columns=":metadata.name" --field-selector="${fields}" -n "${namespace}"| grep -E "${regex}"))

  dump_header "$message"

  if [ -z "$object_names" ] ; then
    log "No resources of object type: \"${type}\" in namespace: \"${namespace}\" with the current specifications were located"
  fi

  for object in "${object_names[@]}"
  do
    log ""
    log "========================================================"
    log "Command: ${command}, type: ${type}, name: ${object}"
    log "========================================================"
    if [ "$command" == "describe" ] ; then
      kubectl "${command}" "${type}" "${object}" -n "${namespace}"
    elif [ "$command" == "logs" ] ; then
      if [ -z "$container" ] ; then
        kubectl "${command}" "${object}" -n "${namespace}"
      else
        kubectl "${command}" "${object}" -n "${namespace}" -c "${container}"
      fi
    fi
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

  echo $(join_by , "${states[@]}")
}


# join an array with a specified value
# $1 join - value to join by
# $2 values - values in which to join
# Usage:
# join_by , "${ARRAY[@]}"
function join_by() {
  local IFS="$1"
  shift
  echo "$*"
}

# prints usage message for this script to consoleerr
# Usage:
# usage
function usage {
    error
    error "usage: $0 -o object_type -n namespace -m message [-r name_regex] [-s state] [-S not_state] [-l] [-c container] [-h]"
    error " -o object_type   Type of the object (i.e. namespaces, pods, jobs, etc)"
    error " -n namespace     Namespace of the given object type"
    error " -r name_regex    Regex to retrieve certain objects by name (Optional)"
    error " -s state         Specified state the described object should be in (i.e. Running) (Multiple values allowed) (Optional)"
    error " -S not_state     Specified state that the described object should not be in (Multiple values allowed) (Optional)"
    error " -m message       Message for the diagnostic header to inform on cause of output"
    error " -l               Retrieve logs for specified object"
    error " -c container     Container in which to pull logs from"
    error " -h               Help"
    error
    exit 1
}

NAMESPACE="default"
NAME_REGEX=""
STATES=()
NOT_STATES=()
MESSAGE=""
COMMAND="describe"
while getopts o:n:r:s:S:m:lc:h flag
do
    case "${flag}" in
        o) OBJECT_TYPE=${OPTARG};;
        n) NAMESPACE=${OPTARG};;
        r) NAME_REGEX=${OPTARG};;
        s) STATES+=("${OPTARG}");;
        S) NOT_STATES+=("${OPTARG}");;
        m) MESSAGE=${OPTARG};;
        l) COMMAND="logs";;
        c) CONTAINER="${OPTARG}";;
        h) usage;;
        *) usage;;
    esac
done
shift $((OPTIND -1))

STATE_FORMAT=$(format_field_selectors "status.phase" "=" "${STATES[@]}")
NOT_STATE_FORMAT=$(format_field_selectors "status.phase" "!=" "${NOT_STATES[@]}")
FIELD_SELECTORS="${STATE_FORMAT},${NOT_STATE_FORMAT}"

dump_objects "${COMMAND}" "${OBJECT_TYPE}" "${NAMESPACE}" "${NAME_REGEX}" "${FIELD_SELECTORS}" "${MESSAGE}" "${CONTAINER}"