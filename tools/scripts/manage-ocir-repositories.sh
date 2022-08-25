#!/bin/bash
# Copyright (c) 2021, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Create a Docker repository in a specific compartment using an exploded tarball of Verrazzano images; useful for
# scoping a repo someplace other than the root compartment.
#
set -o pipefail
set -o errtrace

DRY_RUN=
IMAGES_DIR=.
REGION=""
REGION_SHORT_NAME=""
COMPARTMENT_ID=""
PARENT_REPO=""
USE_BOM=false
CREATE_REPOS=false
DELETE_REPOS=false
DELETE_ALL_REPOS=false
USE_LOCAL_IMAGES=false
INCLUDE_COMPONENTS=
EXCLUDE_COMPONENTS=

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. ${SCRIPT_DIR}/bom_utils.sh

usage() {
  ec=${1:-0}
  echo """
Utility script for creating or deleting OCIR repos for Verrazzano images.

You can create OCIR repos in a valid OCI tenancy compartment using a user-provided repository prefix from either

- a set of exported Docker images tarballs located in a local directory in a specific
- a valid Verrazzano BOM registry file

For the local images case, this script reads the image repo information out of each tar file; in the BOM case it will
retrieve the image information from there.

It then uses the above information to create a corresponding image repo in the target tenancy compartment, under that
tenancy's namespace.

See https://docs.oracle.com/en-us/iaas/Content/Registry/Tasks/registrycreatingarepository.htm for details on
repository creation in an OCI tenancy.

For the deletion case, the script uses the provided compartment ID (-i) and repository prefix (-p) to find all matching
repositories in the compartment and deletes them.

In both cases, you can provide either the full region name (e.g., "us-phoenix-1", or the region short name (e.g., "phx").

Create usage:

$0  -c -p <parent-repo> -i <compartment-ocid> [-r <region-name> | -s <region-code> ] [ -l <path> | -b <path> ] [ -n <component-name> -e <component-name> ]

Delete usage:

$0  -d -p <parent-repo> -i <compartment-ocid> [-r <region-name> | -s <region-code> ]

-a                  Used with -d, delete all repositories in specified compartment matching -p <repo-path>
-b <path>           Bill of materials (BOM) of Verrazzano components; if not specified, defaults to ./verrazzano-bom.json
-c                  Create repository
-d                  Delete repository
-e                  Exclude component (when used with -b)
-f                  Used -with -d, force deletion without prompting
-i <ocid>           Compartment ID
-l <path>           Used with -c, images directory location for creating repos based on saved local image tar files
-n                  Include component (when used with -b)
-p <repo-prefix>    Parent repo, without the tenancy/ObjectStore namespace prefix
-r <region-name>    Region name (e.g., "us-phoenix-1")
-s <region-code>    Region short name (e.g., "phx", "lhr")
-w                  Used with -d, to wait for delete completion
-z                  Dry-run mode; runs without executing OCI repository commands



# Create repos in compartment ocid.compartment.oc1..blah, with the repository prefix "myreporoot/testuser/myrepo/v8o", and the extracted tarball location is /tmp/exploded:
$0 -c -p myreporoot/testuser/myrepo/v8o -r uk-london-1 -i ocid.compartment.oc1..blah -d /tmp/exploded

# Create repos in compartment ocid.compartment.oc1..blah, with the repository prefix myreporoot/testuser/myrepo/v8o using a BOM file
$0 -c -p myreporoot/testuser/myrepo/v8o -r uk-london-1 -i ocid.compartment.oc1..blah -b /tmp/local-bom.json

# Only create repos for Istio and NGINX components
$0 -c -p myreporoot/testuser/myrepo/v8o -r uk-london-1 -i ocid.compartment.oc1..blah -b /tmp/local-bom.json -n istio -n ingress-nginx

# Force-delete all repos starting with myreporoot/testuser/myrepo/v8o in the specified compartment and wait for it to finish
$0 -d -a -p myreporoot/testuser/myrepo/v8o -r uk-london-1 -i ocid.compartment.oc1..blah -f -w

# Create only the cert-manager repos in the specified BOM file and wait for it to finish
$0 -c -r phx -i ocid1.compartment.oc1..aaaaaaaa7cfqxbsnon63unlmm5z63zidx5wvq4gieuc5kixemfitzliwvxeq -p myreporoot/testuser/myrepo/v8o -n cert-manager -b ./master-generated-verrazzano-bom.json

# Force-delete only the cert-manager repos in the specified BOM file and wait for it to finish
$0 -d -r phx -i ocid1.compartment.oc1..aaaaaaaa7cfqxbsnon63unlmm5z63zidx5wvq4gieuc5kixemfitzliwvxeq -p myreporoot/testuser/myrepo/v8o -n cert-manager -b ./master-generated-verrazzano-bom.json -f -w
  """
  exit ${ec}
}

function create_ocir_repo() {
  local repo_path=$1

  local is_public="false"
  if [ "$resolvedRepository" == "rancher" ] || [ "$from_image_name" == "verrazzano-platform-operator" ] \
    || [ "$from_image_name" == "fluentd-kubernetes-daemonset" ] || [ "$from_image_name" == "proxyv2" ] \
    || [ "$from_image_name" == "weblogic-monitoring-exporter" ]; then
    # Rancher repos must be public
    is_public="true"
  fi

  echo "Creating repository ${repo_path} in ${REGION}, public: ${is_public}"

  if [ "${DRY_RUN}" != "true" ]; then
    oci --region ${REGION} artifacts container repository create --display-name ${repo_path} \
      --is-public ${is_public} --compartment-id ${COMPARTMENT_ID}
  else
    echo "Dry run, skipping action..."
  fi
}

function delete_ocir_repo_by_ocid() {
  local id=$1
  repo_name=$(oci --region ${REGION} artifacts container repository get --repository-id ${id} | jq -r '.data."display-name"')
  echo "Deleting repository ${repo_name}, id: ${id}"
  if [ "${DRY_RUN}" != "true" ]; then
    oci --region ${REGION} artifacts container repository delete --repository-id ${id} ${FORCE} ${WAIT_FOR_STATE}
  else
    echo "Dry run, skipping action..."
  fi
}

# Main driver for processing images from a locally downloaded set of tarballs
function process_image_repos_from_archives() {
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
    local resolvedRepository=$(dirname $from_image | cut -d \/ -f 2-)

    local repo_path=${resolvedRepository}/${from_image_name}
    if [ -n "${PARENT_REPO}" ]; then
      repo_path=${PARENT_REPO}/${repo_path}
    fi

    process_image_repo ${repo_path}
  done
}

# Returns 0 if the specified component is in the excludes list, 1 otherwise
function is_component_excluded() {
    local seeking=$1
    local excludes=(${EXCLUDE_COMPONENTS})
    local in=1
    for comp in "${excludes[@]}"; do
        if [[ "$comp" == "$seeking" ]]; then
            in=0
            break
        fi
    done
    return $in
}

function process_image_repo() {
  local repo_path=$1
  if [ "${CREATE_REPOS}" == "true" ]; then
    create_ocir_repo ${repo_path}
  elif [ "${DELETE_REPOS}" == "true" ]; then
    delete_all_repos_for_path ${repo_path}
  fi
}

function process_image_repos_from_bom() {
  # Loop through registry components
  echo "Using image registry ${BOM_FILE}"

  local components=(${INCLUDE_COMPONENTS})
  if [ "${#components[@]}" == "0" ]; then
    components=($(list_components))
  fi

  echo "Components: ${components[*]}"

  for component in "${components[@]}"; do
    if is_component_excluded ${component} ; then
      echo "Component ${component} excluded"
      continue
    fi
    local subcomponents=($(list_subcomponent_names ${component}))
    for subcomp in "${subcomponents[@]}"; do
      echo "Processing images for Verrazzano subcomponent ${component}/${subcomp}"
      # Load the repository and base image names for the component
      #local resolvedRepository=$(get_subcomponent_repo $component $subcomp)
      local image_names=$(list_subcomponent_images $component $subcomp)

      # for each image in the subcomponent list:
      # - resolve the BOM registry location for the image
      # - resolve the BOM repository for the image
      # - build the from/to locations for the image
      # - call process_image to pull/tag/push the image
      for base_image in ${image_names}; do
        local resolvedRegistry=$(resolve_image_registry_from_bom $component $subcomp $base_image)
        local from_image_prefix=${PARENT_REPO}
        local resolvedRepository=$(resolve_image_repo_from_bom $component $subcomp $base_image)
        if [ -n "${resolvedRepository}" ] && [ "${resolvedRepository}" != "null" ]; then
          from_image_prefix=${from_image_prefix}/${resolvedRepository}
        fi

        # Build up the image name and target image name, and do a pull/tag/push
        local imageName=$(echo $base_image | cut -d \: -f 1)
        local repo_path=${from_image_prefix}/${imageName}

        process_image_repo ${repo_path}
      done
    done
  done
}

function delete_all_repos_for_path() {
  repo_prefix=$1

  if [ -z "${repo_prefix}" ]; then
    echo "No repository prefix specified for deletion"
    exit 1
  fi
  repo_ids="$(oci --region ${REGION} artifacts container repository list --compartment-id  ${COMPARTMENT_ID} | \
    jq -r --arg pattern "${repo_prefix}" '.data.items[] | select(."display-name"|test($pattern)) | .id')"

  for id in ${repo_ids}; do
    delete_ocir_repo_by_ocid $id
  done

  return 0
}

while getopts "acdfhwzb:e:i:l:n:p:r:s:" opt; do
  case ${opt} in
  a) # Delete all repos, with -d
    DELETE_ALL_REPOS=true
    ;;
  b) # Create repo
    USE_BOM=true
    BOM_FILE=${OPTARG}
    ;;
  c) # Create repo
    CREATE_REPOS=true
    ;;
  d) # Delete repo
    DELETE_REPOS=true
    ;;
  e)
    echo "Exclude component: ${OPTARG}"
    EXCLUDE_COMPONENTS="${EXCLUDE_COMPONENTS} ${OPTARG}"
    ;;
  f ) # force delete, no prompt
    FORCE="--force"
    ;;
  i) # compartment ID
    COMPARTMENT_ID=${OPTARG}
    ;;
  l) # images dir
    USE_LOCAL_IMAGES=true
    IMAGES_DIR="${OPTARG}"
    ;;
  n)
    echo "Include component: ${OPTARG}"
    INCLUDE_COMPONENTS="${INCLUDE_COMPONENTS} ${OPTARG}"
    ;;
  p) # parent repo
    PARENT_REPO=${OPTARG}
    ;;
  r) # region
    REGION=${OPTARG}
    ;;
  s )
    REGION_SHORT_NAME="${OPTARG}"
    ;;
  w ) # wait for deletion
    WAIT_FOR_STATE="--wait-for-state DELETED"
    ;;
  z) # dry-run
    DRY_RUN=true
    ;;
  h)
    usage 0
    ;;
  *)
    usage 0
    ;;
  esac
done
shift $((OPTIND - 1))

function check() {
  if [ "${CREATE_REPOS}" == "${DELETE_REPOS}" ]; then
    echo "Must specify only one valid operation, only one of -c or -d must be set"
    exit 1
  fi

  if [[ "${DELETE_REPOS}" == "true" ]]; then
    if [[ "${USE_LOCAL_IMAGES}" == "true" && (-n "${INCLUDE_COMPONENTS}" || -n "${EXCLUDE_COMPONENTS}") ]]; then
      echo "Delete repositories, can only use -e or -n with -b (BOM File) option, not -l"
      exit 1
    fi 
  fi

  if [ "${CREATE_REPOS}" == "true" ] && [ "${DELETE_ALL_REPOS}" == "true" ]; then
    echo "Warning: -a (delete all repos) set with -c, ignoring"
    DELETE_ALL_REPOS=false
  fi

  if [[ "${DELETE_REPOS}" == "true" && "${USE_LOCAL_IMAGES}" == "true" ]]; then
    echo "Can not specify -l with -d"
    exit 1
  fi

  if [[ "${DELETE_REPOS}" == "true" && "${USE_BOM}" == "false" && "${DELETE_ALL_REPOS}" == "false" ]]; then
    echo "Delete repostories, must specify exactly one of either -a (all repos) or -b (BOM file)"
    exit 1
  fi

  if [[ "${DELETE_ALL_REPOS}" == "true" && (-n "${INCLUDE_COMPONENTS}" || -n "${EXCLUDE_COMPONENTS}" || "${USE_BOM}" == "true" || "${USE_LOCAL_IMAGES}" == "true") ]]; then
    echo "Can not specify -l, -b, -n, or -e with -a"
    exit 1
  fi

  if [ "${USE_LOCAL_IMAGES}" == "true" ] && [ -z "${IMAGES_DIR}" ]; then
    echo "Use local images specified, but no location specified"
    exit 1
  fi

  if [ -z "${PARENT_REPO}" ]; then
    echo "Repository pattern not provided"
    exit 1
  fi

  if [ -z "${COMPARTMENT_ID}" ]; then
    echo "Compartment ID not provided"
    exit 1
  fi

  echo "Checking if OCI CLI is installed ..."
  if ! oci --help >/dev/null; then
    echo "[ERROR] OCI CLI is not installed, please install"
    exit 1
  fi

  echo "Checking if jq is installed ..."
  if ! jq --help >/dev/null; then
    echo "[ERROR] jq is not install ... please install jq"
    exit 1
  fi

  if [ -z "${REGION}" ]; then
    if [ -z "${REGION_SHORT_NAME}" ]; then
      echo "Must provide either the full or the short region name"
      exit 1
    fi
    echo "REGION_SHORT_NAME=$REGION_SHORT_NAME"
    REGION=$(oci --region us-phoenix-1 iam region list | jq -r  --arg regionAbbr ${REGION_SHORT_NAME} '.data[] | select(.key|test($regionAbbr;"i")) | .name')
    if [ -z "${REGION}" ] || [ "null" == "${REGION}" ]; then
      echo "Invalid short region name ${REGION_SHORT_NAME}"
      exit 1
    fi
  fi
}

function main() {
  if [ "${DELETE_ALL_REPOS}" == "true" ] && [ "${DELETE_REPOS}" == "true" ]; then
    echo "Deleting all OCIR repositories matching the path ${PARENT_REPO}"
    delete_all_repos_for_path ${PARENT_REPO}
  else
    if [ "${USE_LOCAL_IMAGES}" == "true" ]; then
      echo "Processing images from local archives at ${IMAGES_DIR}"
      process_image_repos_from_archives
    elif [ "${USE_BOM}" == "true" ]; then
      echo "Processing images from BOM file ${BOM_FILE}"
      process_image_repos_from_bom
    fi
  fi

  echo "Done."
}

check
main

