#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "$JENKINS_URL" ] || [ -z "$WORKSPACE" ] || [ -z "$OCI_OS_NAMESPACE" ] || [ -z "$OCI_OS_BUCKET" ]; then
  echo "This script must only be called from Jenkins and requires a number of environment variables are set"
  exit 1
fi

if [ ! -f "${WORKSPACE}/verrazzano-bom.json" ]; then
  echo "There is no verrazzano-bom.json from this run, so we can't push anything to OCIR"
  exit 1
fi

# Periodic runs happen much more frequently than master promotions do, so we only conditionally do pushes to OCIR

# If we have a previous last-ocir-pushed-verrazzno-bom.json, then see if it matches the verrazzano-bom.json used
# to test with in this run. If they match, then we have already pushed the images for this verrazzano-bom.json
# into OCIR for Master periodic runs and we do not need to do that again.
# If they don't match, or if we didn't have one to compare, then we will proceed to push them to OCIR
if [ -f "${WORKSPACE}/last-ocir-pushed-verrazzano-bom.json" ]; then
  diff ${WORKSPACE}/last-ocir-pushed-verrazzano-bom.json ${WORKSPACE}/verrazzano-bom.json > /dev/null
  if [ $? -eq 0 ]; then
    echo "OCIR images for this verrazzano-bom.json have already been pushed to OCIR for scanning in a previous periodic run, skipping this step"
    exit 0
  fi
fi

# We should have image tar files created already in ${WORKSPACE}/tar-files
if [ ! -d "${WORKSPACE}/tar-files" ]; then
  echo "No tar files were found to push into OCIR"
  exit 1
fi

# DOCKER LOGIN TO OCIR (do in Jenkinsfile)

export MYREG=iad.ocir.io
export MYREPO=odsbuilddev/sandboxes/tony.vlatas/tony-scan-test-1.0.1
export VPO_IMAGE=$(cat verrazzano-bom.json | jq -r '.components[].subcomponents[] | select(.name == "verrazzano-platform-operator") | "\(.repository)/\(.images[].image):\(.images[].tag)"')

# TBD: It may make more sense to create a combined script for our needs here. One that would do the findOrCreate repo that we need, and also that gives
# us an exact list of the repositories that were created at the same time (ie: no need to get them from OCI later on). We also could integrate in adding
# the scanner before we push any images at all to the repository, so it would all be setup before we push the images in.

# Create repositories, TBD: We really need a find or create here, even with different BOMs most will exist already
sh ~/src/github.com/verrazzano/verrazzano/tests/e2e/config/scripts/create_ocir_repositories.sh -p sandboxes/tony.vlatas/tony-scan-test-1.0.1 -r us-ashburn-1 -c insert-your-sandbox-compartment-id-here -d ${WORKSPACE}/tar-files

# Push the images
sh vz-registry-image-helper.sh -t $MYREG -r $MYREPO -l ${WORKSPACE}/tar-files

# Get a filtered repository list
oci artifacts container repository list --compartment-id ocid1.compartment.oc1..aaaaaaaatx2vnmw2wkcbkefseeym7nsqt4w2ulzlelzzm2dwuobvlcnnuepq --region us-ashburn-1 --all > iad-all-repos.jsoncat iad-all-repos.json | jq '.data.items[]."display-name" | select(test(".*1.0.1.*")?)'

		  CONDITIONAL: Only execute this stage if the last-scanpush-commit.txt doesn't match the last-stable-commit.txt (Actually I think we can use the boms here as well, so only one new file needed)
			- Pull down zip and extract files
			- FindOrCreateRepositories
				- Check each one that is required here
				- I think we always run this, in case new images are added
				- TBD: Removing old ones
			- Checks for existence of repositories
			- Creates repositories based on location
			- adds scanners, etc...
			- Pushes images to OCIR
			- update last-scanpush-commit.txt to be have the current last-stable-commit.txt value for this run
			- Also create a last-scanpush-bom.json, ie: the BOM used. That will be useful for the scan polling job to know exactly which images to look for results for


# Finally push the current verrazzano-bom.json up as the last-ocir-pushed-verrazzano-bom.json so we know those were the latest images
# pushed up. This is used above for avoiding pushing things multiple times for no reason, and it also is used when polling for results
# to know which images were last pushed for Master (which results are the latest)
oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master-last-clean-periodic-test/last-ocir-pushed-verrazzano-bom.json --file ${WORKSPACE}/verrazzano-bom.json


# TBD: We could also save the list of repositories as well, that may save the polling job some work so it doesn't need to figure that out 
