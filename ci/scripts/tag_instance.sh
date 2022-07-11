#!/usr/bin/env bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

# The branch names which we put into the tag are: master, release-*, and feature
# We do not differentiate between feature branches currently, the values of these change frequently and the overall
# usage related to feature branch testing is good enough granularity to start with here.
if [ -z "$env.BRANCH_NAME" ]; then
  echo "Script needs to be run in Jenkins environment where env.BRANCH_NAME is defined"
  exit 1
fi
if [[ $env.BRANCH_NAME =~ '/^release-.*/' ]] || [[ $env.BRANCH_NAME == 'master' ]]; then
  CURRENT_BRANCH_NAME=$env.BRANCH_NAME
else
  CURRENT_BRANCH_NAME="feature-branch"
fi

# The pipeline names we put into the tag are: none, main, periodic, release, scan, oscs, bfs, doc, ...)
# These get set/propagated, "none" generally means that the parameter was not set/propagated for the job run,
# for example if a developer is running a test job in isolation they would not set the parameter and we
# can differentiate those runs from full pipeline runs (generally it is more efficient when developers test
# the explicit jobs that need testing versus running full sets of tests).
# This tag will help us to differentiate when a job scenario for a branch is being run for a different
# end goal (ie: same job scenario/branch could be run in the push triggered or periodics, etc...).
# For some this is somewhat redundant as the jobscenario may be enough to identify when it is run (scan, oscs, bfs)
# but having it defined here may help simplify the differentiation for dashboard purposes
if [ -z "$1" ]; then
  echo "Pipeline name must be specified"
  exit 1
fi
CURRENT_PIPELINE="$1"

# This is used to identify the Job Scenario: jobname+scenario
# The intention here is that we can differentiate between a job being run with different parameters
# in order to test different variants. For example, KindAT job run using specific K8S version, using Calico, etc...
# The allowed names are defined on a wiki (or maybe in source control?).
# A developer running it ad-hoc may set the scenario name, or not. We could try to get fancy and determine
# the scenario based on parameter values (ie: have the job determine it rather than be told it). The POC
# is being told it.
# NOTE: We want to be able to differentiate these, and these are most likely to be changing over time as
# Job names change and what we test changes.
if [ -z "$2" ]; then
  echo "Test job scenario must be specified"
  exit 1
fi
CURRENT_JOB_SCENARIO="$2"

# Get the instance OCID (note that the grep is being used to trim it)
INSTANCE_OCID=$(curl -s http://169.254.169.254/opc/v1/instance | jq '.id' | grep -o -e '[a-z, 0-9, A-Z, \.]*')

# Add as free-form tags in the POC. We should define a namespace and use defined-tags if the POC progresses
# to be an actual implementation.
oci compute instance update --force --instance-id $INSTANCE_OCID --freeform-tags "{ \"verrazzano-infra/Branch\" : \"$CURRENT_BRANCH_NAME\", \"verrazzano-infra/Pipeline\" : \"$CURRENT_PIPELINE\", \"verrazzano-infra/JobScenario\" : \"$CURRENT_JOB_SCENARIO\"}"
