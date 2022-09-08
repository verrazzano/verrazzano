#!/bin/bash
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Create a Docker repository in a specific compartment using an exploded tarball of Verrazzano images; useful for
# scoping a repo someplace other than the root compartment.
#
set -u

usage() {
  echo """
Create OCIR repos for a set of exported Docker images tarballs located in a local directory in a specific
tenancy compartment.  This script reads the image repo information out of each tar file and uses that to create
a corresponding Docker repo in the target tenancy compartment, under that tenancy's namespace.  You can provide either
the full region name (e.g., "us-phoenix-1", or the region short name (e.g., "phx").

See https://docs.oracle.com/en-us/iaas/Content/Registry/Tasks/registrycreatingarepository.htm for details on
repository creation in an OCI tenancy.

Usage:

$0  -p <parent-repo> -c <compartment-id> [ -r <region> -d <path> ]

-r Region name (e.g., "us-phoenix-1")
-s Region short name (e.g., "phx", "lhr")
-p Parent repo, without the tenancy namespace
-c Compartment ID
-d Images directory
-t Target ID. This is an existing Target-Id used for OCIR scanning. Repositories which are not already targeted there
   will be created as PRIVATE-ONLY and will also added to the target list for this existing Target Id. If not found
   this will fail. Note that repositories which are already targeted will be skipped for creation (they already exist).
   This is used by the CI for OCIR scanning setup.

Example, to create a repo in compartment ocid.compartment.oc1..blah, where the desired docker path with tenancy namespace
to the image is "myreporoot/testuser/myrepo/v8o", and the extracted tarball location is /tmp/exploded:

$0 -p myreporoot/testuser/myrepo/v8o -r uk-london-1 -c ocid.compartment.oc1..blah -d /tmp/exploded
  """
}

# Function to check the env for OCIR scan related functions
function checkEnvForScan() {
  if [ -z $OCIR_SCAN_TARGET_ID ]; then
    echo "OCIR_SCAN_TARGET_ID is required to be defined"
    return 1
  fi

  if [ -z $REGION ]; then
    echo "REGION is required to be defined"
    return 1
  fi
  return 0
}

# getTargetJson
function getTargetJson() {
  checkEnvForScan
  if [ $? -ne 0 ]; then
    return 1
  fi

  if [ -z $1 ]; then
    echo "Please specify the name of the environment variable to return the target json filename into"
    return 1
  fi
  local  __resultvar=$1

  local targetfile=$(mktemp temp-target-XXXXXX.json)
  oci vulnerability-scanning container scan target get --container-scan-target-id $OCIR_SCAN_TARGET_ID --region ${REGION} > $targetfile
  if [ $? -ne 0 ]; then
    echo "Failed to get target $OCIR_SCAN_TARGET_ID in $REGION"
    return 1
  fi

  # Set the target output filename into the supplied environment variable
  eval $__resultvar="'$targetfile'"
  return 0
}

# addNewRepositoriesToTarget:  This will update the existing target by adding new repositories into it
#   $1 array of repositories to add
#   $2 target.json
function addNewRepositoriesToTarget() {
  checkEnvForScan
  local retval=0
  if [ $? -ne 0 ]; then
    return 1
  fi

  if [ -z $1 ] || [ -z $2 ] || [ ! -f $2 ]; then
    echo "Invalid arguments supplied to addNewRepositoriesToTarget"
    return "error"
  fi

  # REVIEW: If we can give the array to jq all at once that will be nice, but for now
  # just adding one element at a time here
  local newtargetfile=$(mktemp temp-new-target-XXXXXX.json)
  cp $2 $newtargetfile
  read -ra repo_array <<< $1
  for repo_name in "${repo_array[@]}"
  do
    echo $(jq --arg repo "$repo_name" '.data."target-registry".repositories += [$repo]' $newtargetfile | jq '.') > $newtargetfile
    if [ $? -ne 0 ]; then
      echo "Problem adding new repositories into the existing Target"
      retval=1
    fi
  done

  if [ $retval -eq 0 ]; then
    cat $newtargetfile
    # Update the target using the new target file
    oci vulnerability-scanning container scan target update --from-json file://$newtargetfile --container-scan-target-id $OCIR_SCAN_TARGET_ID --region ${REGION}
    if [ $? -ne 0 ]; then
      echo "Problem updating the Target"
      retval=1
    fi
  fi
  rm $newtargetfile
  return $retval
}

# isRepositoryTargeted. returns true if found in the list, false if not found, and error if arguments are invalid
#  $1 name of repository
#  $2 target.json
function isRepositoryTargeted() {
  if [ -z $1 ] || [ -z $2 ] || [ ! -f $2 ]; then
    echo "Invalid arguments supplied to isRepositoryTargeted"
    return 2
  fi
  grep $1 $2 > /dev/null
  return $?
}

# Main driver for processing images from a locally downloaded set of tarballs
function create_image_repos_from_archives() {
  declare -a added_repositories=()
  local target_file=""

  # If we have a scan target, and if it is usable.
  # NOTE: We are having issues with the scan target getting into a bad state, the lookup can start failing
  # with a 404. If that happens we proceed without it and do NOT fail.
  local target_accessed="false"
  if [ ! -z $OCIR_SCAN_TARGET_ID ]; then
    getTargetJson "target_file"
    if [ $? -eq 0 ]; then
      echo "Target JSON was retrieved"
      target_accessed="true"
    else
      echo "No target JSON was retrieved, target related operations will be skipped but other processing will proceed based on target id being specified"
    fi
  fi

  local repositories_listed="false"
  local reposfile=$(mktemp temp-repositories-XXXXXX.json)
  oci --region ${REGION} artifacts container repository list --compartment-id ${COMPARTMENT_ID} --all > $reposfile
  if [ $? -eq 0 ]; then
    echo "Repositories listed"
    repositories_listed="true"
  else
    echo "Unable to list the existing repositories to check for existence"
  fi

  # Loop through tar files
  echo "Using local image downloads"
  for file in ${IMAGES_DIR}/*.tar; do
    echo "Processing file ${file}"
    if [ ! -e ${file} ]; then
      echo "Image tar file ${file} does not exist!"
      exit 1
    fi

    # Build up the image name and target image names, and create the repo
    local from_image=$(tar xOvf $file manifest.json | jq -r '.[0].RepoTags[0]')
    local from_image_name=$(basename $from_image | cut -d \: -f 1)
    local from_repository=$(dirname $from_image | cut -d \/ -f 2-)

    # When OCIR_SCAN_TARGET_ID is set, all of the repositories used for scanning are created as private (we only use them for scanning)
    local is_public="false"
    if [ "$from_repository" == "rancher" ] || [ "$from_image_name" == "verrazzano-platform-operator" ] \
      || [ "$from_image_name" == "fluentd-kubernetes-daemonset" ] || [ "$from_image_name" == "proxyv2" ] \
      || [ "$from_image_name" == "weblogic-monitoring-exporter" ] && [ -z $OCIR_SCAN_TARGET_ID ]; then
      # Rancher repos must be public
      is_public="true"
    fi

    local repo_path=${from_repository}/${from_image_name}
    if [ -n "${PARENT_REPO}" ]; then
      repo_path=${PARENT_REPO}/${repo_path}
    fi

    # If we have a scan target, we can see if it is targeted already (if it is we can skip creating it)
    # if not we track the ones we go ahead and create so we can target them afterwards
    if [ ! -z $OCIR_SCAN_TARGET_ID ]; then
      # Only try target related operations if we were able to access the target, skip otherwise
      if [ "$target_accessed" == "true" ]; then
        isRepositoryTargeted $repo_path $target_file
        local repository_targeted=$?
        if [ $repository_targeted -eq 2 ]; then
          echo "Error checking if repository was targeted ${repo_path}"
          rm $target_file
          exit 1
        fi
        # If it is targeted already, then it exists and we skip creating a new repository
        if [ $repository_targeted -eq 0 ]; then
          echo "$repo_path is already targeted"
          continue
        fi
        # If we got here, we will add it to the list to try to target it
        added_repositories+=("$repo_path")
        echo "$repo_path needs to be targeted"
      else
        echo "skipping target checking for $repo_path (target not accessible)"
      fi

      # Check if it exists already first
      if [ "$repositories_listed" == "false" ]; then
        echo "skipping existing repository check for $repo_path (repositories unable to be listed)"
        continue
      fi

      grep "${repo_path}" $reposfile > /dev/null
      if [ $? -eq 0 ]; then
        echo "$repo_path already exists and doesn't need to be created"
        continue
      fi
      echo "$repo_path needs to be created"
    fi

    echo "Creating repository ${repo_path} in ${REGION}, public: ${is_public}"
    oci --region ${REGION} artifacts container repository create --display-name ${repo_path} \
      --is-public ${is_public} --compartment-id ${COMPARTMENT_ID}
  done

  # If we added new repositories, we need to get them added to the target so they will get scanned
  if [ ! -z $OCIR_SCAN_TARGET_ID ] && [ "$target_accessed" == "true" ]; then
    # FIXME: Do not enable until we are sure the VSS lifecycle state issues with update are understood and handled
    #  addNewRepositoriesToTarget "${added_repositories}" $target_file
    rm $target_file || true
  fi

  if [ "$repositories_listed" == "true" ]; then
    rm $reposfile || true
  fi
}

IMAGES_DIR=.
REGION=""
REGION_SHORT_NAME=""
OCIR_SCAN_TARGET_ID=""
COMPARTMENT_ID=""
PARENT_REPO=""

while getopts ":s:c:r:p:d:t:" opt; do
  case ${opt} in
  c) # compartment ID
    COMPARTMENT_ID=${OPTARG}
    ;;
  r) # region
    REGION=${OPTARG}
    ;;
  p) # parent repo
    PARENT_REPO=${OPTARG}
    ;;
  d) # images dir
    IMAGES_DIR="${OPTARG}"
    ;;
  s )
    REGION_SHORT_NAME="${OPTARG}"
    ;;
  t )
    OCIR_SCAN_TARGET_ID="${OPTARG}"
    ;;
  \?)
    usage
    ;;
  esac
done
shift $((OPTIND - 1))

if [ -z "${REGION}" ]; then
  if [ -z "${REGION_SHORT_NAME}" ]; then
    echo "Must provide either the full or the short region name"
    usage
    exit 1
  fi
  echo "REGION_SHORT_NAME=$REGION_SHORT_NAME"
  REGION=$(oci --region us-phoenix-1 iam region list | jq -r  --arg regionAbbr ${REGION_SHORT_NAME} '.data[] | select(.key|test($regionAbbr;"i")) | .name')
  if [ -z "${REGION}" ] || [ "null" == "${REGION}" ]; then
    echo "Invalid short region name ${REGION_SHORT_NAME}"
    usage
    exit 1
  fi
fi

if [ -z "${PARENT_REPO}" ]; then
  echo "Repository pattern not provided"
  usage
  exit 1
fi
if [ -z "${COMPARTMENT_ID}" ]; then
  echo "Compartment ID not provided"
  usage
  exit 1
fi

create_image_repos_from_archives
