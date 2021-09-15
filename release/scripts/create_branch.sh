#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage {
    echo
    echo "usage: $0 [-v release_version] [-c source_commit]"
    echo "  -v release_version         The release version (e.g. #.# for a major release, #.#.# for a patch release. Required"
    echo "  -c source_commit           The commit hash for the source commit.  Required."
    echo "  -h                         Help"
    echo
    exit 1
}

function is_in_remote() {
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
EXPECTED_SOURCE_BRANCH="master"
EXPECTED_SOURCE_REPO="test"

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
BRANCH=release-${MAJOR}.${MINOR}

# if this is a patch release skip branch creation
if [ "${PATCH}" != "" ]; then
  echo "This is a patch release. No branch creation required"
  exit 0
else
  if ! is_in_remote ${BRANCH} ; then
    echo "creating branch"
    # ensure we are branching off of a verrazzano master branch
    CURRENT_REPO=$(basename `git rev-parse --show-toplevel`)
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

    if [ "${CURRENT_REPO}" != "${EXPECTED_SOURCE_REPO}" ]; then
      echo "Not in the correct repo"
      exit 1
    fi

    if [ "${CURRENT_BRANCH}" != "${EXPECTED_SOURCE_BRANCH}" ]; then
      echo "Not using the master branch as the source branch.  Please checkout the master branch and make sure to pull the latest code"
      exit 1
    fi

    # check that the local branch is up to date
    git fetch
    HEAD=$(git rev-parse HEAD)
    UPSTREAM=$(git rev-parse @{u})
    if [ "${HEAD}" != "${UPSTREAM}" ]; then
      echo "Branch is not up to date.  Performing a pull"
      git pull
    fi

    git checkout -b ${BRANCH} ${RELEASE_COMMIT}
    git push origin ${BRANCH}
  else
    echo "Release branch exists"
    exit 0
  fi
fi




