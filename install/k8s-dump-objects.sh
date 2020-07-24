#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

. $SCRIPT_DIR/common.sh

function dump_header() {
  echo "================================  DIAGNOSTIC OUTPUT START ================================="
  echo ""
}

function dump_footer() {
  echo ""
  echo "================================  DIAGNOSTIC OUTPUT END ==================================="
}

# Dump the objects of the specified type in the given namespace
# $1 Object type
# $2 Namespace - the namespace of the object
#
# Usage:
# dump_objects "pods" "verrazzano-system"
function old_dump_objects () {
  dump_header
  
  local oType=$1
  local ns=$2

  echo ${oType}s 'in namespace' $ns
  echo "--------------------------------------------------------"
  kubectl get $oType -n $ns

  echo ""
  echo 'Describing each '${oType}' in namespace' $ns
  echo "========================================================"
  objs=($(kubectl get $oType -n $ns |  awk 'NR>1 { printf sep $1; sep=" "}'))
  for i in "${objs[@]}"
  do
     echo ""
     echo "--------------------------------------------------------"
     echo  $oType $i
     echo "--------------------------------------------------------"
     kubectl describe $oType -n $ns $i
  done

  dump_footer
}

# Dump the pods in the given namespace
# $1 Namespace - the namespace of the pod
# Usage:
# dump_pods "verrazzano-system"
function dump_pods () {
  old_dump_objects "pod" $1
}

# Dump the jobs in the given namespace
# $1 Namespace - the namespace of the job
# Usage:
# dump_jobs "verrazzano-system"
function dump_jobs () {
  old_dump_objects "job" $1
}

# Dump specified job
# $1 namespace - the namespace of the job
# $2 job regex - regex that will collect the job name from the list of jobs
# Usage:
# dump_job "namespace" "jobRegex"
function dump_job () {
  local jobName=$(kubectl get jobs -n $1 | grep -Eo $2)

  echo ""
  echo "Describing Job for ocrtest: ${jobName}"
  echo "========================================================"
  kubectl describe job ${jobName}
  echo "========================================================"
  echo ""
}

# Dump specified pod
# $1 namespace - the namespace of the pod
# $2 pod regex - regex that will collect the pod name from the list of pods
# Usage:
# dump_pod "namespace" "podRegex"
function dump_pod () {
  local podName=$(kubectl get pods -n $1 | grep -Eo $2)

  echo ""
  echo "Describing Pod for ocrtest: ${podName}"
  echo "========================================================"
  kubectl describe pod ${podName}
  echo "========================================================"
  echo ""
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
    echo "Object type and namespace must be specified to describe objects."
    exit
  fi


  local object_names=($(kubectl get "${type}" --no-headers -o custom-columns=":metadata.name" --field-selector="${fields}" -n "${namespace}"| grep -E "${regex}"))

  for object in "${object_names[@]}"
  do
    echo ""
    echo "========================================================"
    echo "Describing type: ${type}, name: ${object}"
    echo "========================================================"
    kubectl describe "${type}" "${object}" -n "${namespace}"
  done
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
    states+=("${1}${2}${state}")
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
    consoleerr
    consoleerr "usage: $0 -o object_type -n namespace [-r name_regex] [-s state] [-S not_state] [-h]"
    consoleerr " -o object_type   Type of the object (i.e. namespaces, pods, jobs, etc)"
    consoleerr " -n namespace     Namespace of the given object type"
    consoleerr " -r name_regex    Regex to retrieve certain jobs by name"
    consoleerr " -s state         Specified state the described object should be in (i.e. Running)"
    consoleerr " -S not_state     Specified state that the described object should not be in"
    consoleerr " -h               Help"
    consoleerr
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