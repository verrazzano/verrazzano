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

# Dump the ocrtest pod and job
# Usage:
# $1 ocrtest pod name
# dump_ocrtest ${ocrPodName}
function dump_ocrtest () {
  dump_header

  local ocrName = $1

  echo 'Describing Pod for ocrtest: '${ocrName}
  echo "========================================================"
  kubectl describe pod ${ocrName}
  echo "========================================================"

  echo ""
  echo 'Describing Job for ocrtest: '${ocrName}
  echo "========================================================"
  kubectl describe job ocrtest
  echo "========================================================"

  dump_footer
}