#!/usr/bin/env bash
#
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# this attempts to detect if someone has made an update and forgot to run make generate
# if so, it dumps out the status and diffs in the output so you can see what the issue was
# we purposely cause the build to fail in this situation - we do not want the build to
# rely on generated files that are not version controlled.

if [[ -n $(git status --porcelain) ]]; then
  git status
  git diff
  echo "****************************************************************************************************************"
  echo "* ERROR: The result of a 'make generate', 'make manifests' or 'make copyright-check' resulted in files being   *"
  echo "*        modified. These changes need to be included in your PR.                                               *"
  echo "****************************************************************************************************************"
  exit 1
fi