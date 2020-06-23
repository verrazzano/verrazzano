#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
CHECK_VALUES=false
set +u
DNS_TYPE=${DNS_TYPE:-xip.io}
DNS_SUFFIX=${OCI_DNS_ZONE_NAME:-}
# check for valid DNS type
if [ $DNS_TYPE != "xip.io" ] && [ $DNS_TYPE != "oci" ]; then
   echo "DNS_TYPE environment variable  $DNS_TYPE must be set to either oci or xip.io"
   CHECK_VALUES=true
fi
# check for name
if [ $DNS_TYPE = "oci" ]; then
  if [ -z "$VERRAZZANO_ENV_NAME" ]; then
    echo "VERRAZZANO_ENV_NAME environment variable must set for DNS_TYPE=oci"
    CHECK_VALUES=true
  fi
fi
# check expected dns suffix for given dns type
if [ -z "$DNS_SUFFIX" ]; then
  if [ $DNS_TYPE = "oci" ]; then
    echo "OCI_DNS_ZONE_NAME environment variable must set to OCI DNS Zone Name for DNS_TYPE=oci"
    CHECK_VALUES=true
  fi
else
  if [ $DNS_TYPE = "xip.io" ]; then
    echo "OCI_DNS_ZONE_NAME environment variable should not be given with DNS_TYPE=xip.io"
    CHECK_VALUES=true
  fi
fi
set -eu
if [ $CHECK_VALUES = true ]; then
    exit 1
fi
