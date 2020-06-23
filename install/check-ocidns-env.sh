#
# Copyright (c) 2020, Oracle Corporation and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
CHECK_VALUES=false
set +u
if [ -z "$VERRAZZANO_ENV_NAME" ]; then
    echo "VERRAZZANO_ENV_NAME environment variable must set with DNS_TYPE oci"
    CHECK_VALUES=true
fi
if [ -z "$OCI_REGION" ]; then
    echo "OCI_REGION environment variable must set to OCI Region"
    CHECK_VALUES=true
fi
if [ -z "$OCI_TENANCY_OCID" ]; then
    echo "OCI_TENANCY_OCID environment variable must set to OCI Tenancy OCID"
    CHECK_VALUES=true
fi
if [ -z "$OCI_USER_OCID" ]; then
    echo "OCI_USER_OCID environment variable must set to OCI User OCID"
    CHECK_VALUES=true
fi
if [ -z "$OCI_DNS_ZONE_COMPARTMENT_OCID" ]; then
    echo "OCI_DNS_ZONE_COMPARTMENT_OCID environment variable must set to OCI Compartment OCID"
    CHECK_VALUES=true
fi
if [ -z "$OCI_FINGERPRINT" ]; then
    echo "OCI_FINGERPRINT environment variable must set to OCI Fingerprint"
    CHECK_VALUES=true
fi
if [ -z "$OCI_PRIVATE_KEY_FILE" ]; then
    echo "OCI_PRIVATE_KEY_FILE environment variable must set to OCI Private Key File"
    CHECK_VALUES=true
fi
if [ -z "$EMAIL_ADDRESS" ]; then
    echo "EMAIL_ADDRESS environment variable must set to your email address"
    CHECK_VALUES=true
fi
if [ -z "$OCI_DNS_ZONE_OCID" ]; then
    echo "OCI_DNS_ZONE_OCID environment variable must set to OCI DNS Zone OCID"
    CHECK_VALUES=true
fi
if [ -z "$OCI_DNS_ZONE_NAME" ]; then
    echo "OCI_DNS_ZONE_NAME environment variable must set to OCI DNS Zone Name"
    CHECK_VALUES=true
fi
if [ $CHECK_VALUES = true ]; then
    exit 1
fi

[ ! -f $OCI_PRIVATE_KEY_FILE ] && { echo $OCI_PRIVATE_KEY_FILE does not exist; exit 1; }

set -eu
