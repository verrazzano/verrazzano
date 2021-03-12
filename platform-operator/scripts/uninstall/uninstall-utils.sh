#!/bin/bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname $BASH_SOURCE); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../install

. $INSTALL_DIR/common.sh

# wrapper for xargs
# $@ commands - xargs flags and commands
# Usage:
# xargsr commands
function xargsr() {
  unameout=$(uname)
  case "${unameout}" in
    Darwin*)  override=false;;
    FreeBSD*) override=false;;
    *) override=true
  esac
  if "${override}" ; then
    xargs -r "$@"
  else
    xargs "$@"
  fi
}

# error return command
# $1 exit_code - code given by the command that triggered this function
# $2 message - error message given to the user when an error is reached
# Usage:
# err_return $? "message"
function err_return() {
  exit_code=$1
  message=$2

  if (($exit_code < 0 && $exit_code > 255)) ; then
    error "the exit code given is not a valid integer"
    exit 1
  fi

  error "$message"
  return "$exit_code"
}

# Deletes kubernetes resources from all namespaces
# $1 resource-type - type of the resources being deleted
function delete_k8s_resource_from_all_namespaces() {
  local res=$1
  if kubectl get crd "${res}"> /dev/null 2>&1 ; then
    IFS=$'\n' read -r -d '' -a namespaces < <( kubectl get namespaces --no-headers -o custom-columns=":metadata.name" && printf '\0' )
    for ns in "${namespaces[@]}" ; do
      if ! kubectl delete "${res}" --namespace ${ns} --all > /dev/null 2>&1 ; then
        log "Failed to delete ${res} from namespace ${ns}"
      fi
    done
  fi
  # Delete the CRDs, without any CRs based on that, from all the namespaces
  if kubectl get crd "${res}"> /dev/null 2>&1 ; then
    kubectl delete crd "${res}" --ignore-not-found > /dev/null 2>&1
  fi
}


# utility function to delete kubernetes resources
# $1 resource-type - type of the resources being deleted
# $2 custom-cols   - custom columns used when getting the resources
# $3 err-msg       - the message to log if an error is encountered
# $4 filter-args   - arguments given to awk to filter the resource list for deletion (optional)
# $5 namespace     - namespace for resources to be deleted (optional)
# Usage:
# delete_k8s_resources resource-type custom-cols [filter-args] [namespace]
function delete_k8s_resources() {
  ( [ -z "$5" ] && kubectl get $1 --no-headers -o custom-columns="$2" || kubectl get $1 --no-headers -o custom-columns="$2" -n "$5" ) \
    | ( [ -z "$4" ] && cat || awk "$4" ) \
    | ( [ -z "$5" ] && xargsr kubectl delete $1 || xargsr kubectl delete $1 -n "$5" ) \
    || err_return $? "$3" || return $? # return on pipefail
}

# utility function to patch kubernetes resources
# $1 resource-type - type of the resources being patched
# $2 custom-cols   - custom columns used when getting the resources
# $3 err-msg       - the message to log if an error is encountered
# $4 filter-args   - arguments given to awk to filter the resource list for patching
# $5 patch         - the patch to be applied to the resources
# Usage:
# patch_k8s_resources resource-type custom-cols filter-args patch
function patch_k8s_resources() {
  if [ -z "$5" ]; then
    err_return 1 "The patch argument cannot be empty." || return $?
  fi
  kubectl get $1 --no-headers -o custom-columns="$2" \
    | awk "$4" \
    | xargsr kubectl patch $1 -p "$5" --type=merge \
    || err_return $? "$3" || return $? # return on pipefail
}
