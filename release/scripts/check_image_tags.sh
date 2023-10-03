#!/usr/bin/env bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# Checks BOM image tags for problems

usage() {
    cat <<EOM
  Checks BOM image tags for problems.

  Usage:
    $(basename $0) <path to Verrazzano BOM file>

EOM
    exit 0
}

[ -z "$1" -o "$1" == "-h" ] && { usage; }

BOM_FILE=$1
MISMATCH=false

if [ -f $BOM_FILE ]; then
  echo Checking rancher and rancher-agent image tags in $BOM_FILE
  RANCHER_IMAGE_TAG=$(jq -r '.components[].subcomponents[] | select(.name == "rancher") | .images[] | select(.image == "rancher") | "\(.tag)"' $BOM_FILE)
  RANCHER_AGENT_IMAGE_TAG=$(jq -r '.components[].subcomponents[] | select(.name == "rancher") | .images[] | select(.image == "rancher-agent") | "\(.tag)"' $BOM_FILE)
  if [ $RANCHER_IMAGE_TAG != $RANCHER_AGENT_IMAGE_TAG ]; then
    echo Rancher image tag "$RANCHER_IMAGE_TAG" does not match Rancher agent image tag "$RANCHER_AGENT_IMAGE_TAG"
    MISMATCH=true
  fi

  echo Checking fleet and fleet-agent image tags in $BOM_FILE
  FLEET_IMAGE_TAG=$(jq -r '.components[].subcomponents[] | select(.name == "additional-rancher") | .images[] | select(.image == "rancher-fleet") | "\(.tag)"' $BOM_FILE)
  FLEET_AGENT_IMAGE_TAG=$(jq -r '.components[].subcomponents[] | select(.name == "additional-rancher") | .images[] | select(.image == "rancher-fleet-agent") | "\(.tag)"' $BOM_FILE)
  if [ $FLEET_IMAGE_TAG != $FLEET_AGENT_IMAGE_TAG ]; then
    echo Fleet image tag "$FLEET_IMAGE_TAG" does not match Fleet agent image tag "$FLEET_AGENT_IMAGE_TAG"
    MISMATCH=true
  fi

  echo Checking Jaeger image tags in $BOM_FILE
  # All Jaeger images (excluding the jaeger-operator image) should have the same tag
  UNIQUE_JAEGER_IMAGE_TAGS=$(jq -r '.components[] | select(.name == "jaeger-operator") | .subcomponents[] | select(.name != "jaeger-operator") | .images[] | .tag' $BOM_FILE | sort | uniq)
  UNIQUE_JAEGER_IMAGE_TAGS_COUNT=$(echo "$UNIQUE_JAEGER_IMAGE_TAGS" | wc -l)
  if [ $UNIQUE_JAEGER_IMAGE_TAGS_COUNT -gt 1 ]; then
    echo "Expected all Jaeger image tags (excluding jaeger-operator) to match but found these unique tags:"
    echo "$UNIQUE_JAEGER_IMAGE_TAGS"
    MISMATCH=true
  fi

  if [ $MISMATCH == true ]; then
    echo FATAL: One or more mismatched image tags found
    exit 1
  fi
else
  echo FATAL: BOM file $BOM_FILE does not exist
  exit 1
fi
