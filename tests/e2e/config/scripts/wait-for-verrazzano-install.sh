#!/usr/bin/env bash
#
# Copyright (c) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

SECONDS=0
# Each attempt takes roughly ~5 seconds to check, so this should wait not much longer than 1hr for a successful install
# 5 seconds * 12 * 60 = 1hr
MAX_ATTEMPTS=$((12 * 60))
ATTEMPT=0
retval_success=1
retval_failed=1
while [[ $retval_success -ne 0 ]] && [[ $retval_failed -ne 0 ]]; do
  ATTEMPT=$((ATTEMPT+1))
  sleep 5
  output=$(kubectl wait --for=condition=InstallFailed verrazzano/my-verrazzano --timeout=0 2>&1)
  retval_failed=$?
  output=$(kubectl wait --for=condition=InstallComplete verrazzano/my-verrazzano --timeout=0 2>&1)
  retval_success=$?

  if [[ $ATTEMPT -gt $MAX_ATTEMPTS ]]; then
      echo "Timed out waiting for the Verrazzano installation to enter a finished state"
      exit 1
  fi
done

if [ $retval_failed -eq 0 ]; then
    echo "Installation Failed"
    exit 1
fi

echo "Installation completed.  Wait time: $SECONDS seconds"