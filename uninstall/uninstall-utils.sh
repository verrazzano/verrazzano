#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
INSTALL_DIR=$SCRIPT_DIR/../../install

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