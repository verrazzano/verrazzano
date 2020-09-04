#!/bin/bash
#
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

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