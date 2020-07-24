#!/usr/bin/env bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

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
function dump_objects () {
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
  dump_objects "pod" $1
}

# Dump the jobs in the given namespace
# $1 Namespace - the namespace of the job
# Usage:
# dump_jobs "verrazzano-system"
function dump_jobs () {
  dump_objects "job" $1
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
# $1 pod regex - regex that will collect the pod name from the list of pods
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