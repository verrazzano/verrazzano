#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage {
    echo
    echo "usage: $0 [-v release_version] [-c source_commit]"
    echo "  -v release_version         The release version in major_version.minor_version.patch_version format (e.g. 1.1.0). Required"
    echo "  -c source_commit           The commit hash for the source commit.  Required."
    echo "  -h                         Help"
    echo
    exit 1
}

function is_in_remote_repo() {
    local branch=${1}
    local exists=$(git ls-remote --heads origin ${branch})

    if [[ ! -z ${exists} ]]; then
        return 0
    else
        return 1
    fi
}

VERSION=""
RELEASE_COMMIT=""
EXPECTED_SOURCE_BRANCH="origin/master"
RELEASE_BRANCH=""

while getopts v:c:h flag
do
    case "${flag}" in
        v) VERSION=${OPTARG};;
        c) RELEASE_COMMIT=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

parts=( ${VERSION//./ } )
MAJOR="${parts[0]}"
MINOR="${parts[1]}"
PATCH="${parts[2]}"

if [ "${TEST_RUN:-false}" == "false" ]; then
  RELEASE_BRANCH=release-${MAJOR}.${MINOR}
else
  RELEASE_BRANCH=mock-release-${MAJOR}.${MINOR}
fi

echo "Release Branch: ${RELEASE_BRANCH}"

# if this is a patch release skip branch creation
if [ "${PATCH}" != "0" ]; then
  echo "This is a patch release. No branch creation required"
  exit 0
else
  if ! is_in_remote_repo ${RELEASE_BRANCH} ; then
    echo "creating branch"
    # ensure we are branching off of a verrazzano master branch
    
    # check remote repo
    COMMIT_REPO=$(basename `git config --get remote.origin.url`)
    echo "Commit Repo: ${COMMIT_REPO}"
    COMMIT_BRANCH=$(git branch -r --contains ${RELEASE_COMMIT}  | tr -d '[:space:]')
    echo "Remote commit branch: ${COMMIT_BRANCH}"

    if ! [[ "${COMMIT_REPO}" =~ ^(verrazzano|verrazzano.git)$ ]]; then
      echo "Not in the correct repo"
      exit 1
    fi

    # for testing purposes skip branch check
    if [ "${TEST_RUN}" == 'false' ]; then
      if [ "${COMMIT_BRANCH}" != "${EXPECTED_SOURCE_BRANCH}" ]; then
        echo "Not using the master branch as the source branch.  Please checkout the master branch and make sure to pull the latest code"
        exit 1
      fi
    fi

    git checkout -b ${RELEASE_BRANCH} ${RELEASE_COMMIT}
    git push origin ${RELEASE_BRANCH}
  else
    echo "Release branch exists"
    CREATION_COMMIT=$(git rev-parse --verify origin/${RELEASE_BRANCH})
    if [ "${CREATION_COMMIT}" != "${RELEASE_COMMIT}" ]; then
      echo "Release branch found but commits do not match:  Branch HEAD commit=${CREATION_COMMIT}, Requested commit=${RELEASE_COMMIT}"
      exit 1
    fi
    exit 0
  fi
fi




