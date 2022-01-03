#!/bin/bash
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# Script to create a project in Harbor. The script checks whether the specified project already exists in Harbor.
# If it does not exist, it proceeds to create one.

set -u

# variables
REST_API_BASE_URL=
USERNAME=
PASSWORD=
PROJECT_NAME=
IS_PUBLIC=

function usage() {
    echo """
Script to create a project in Harbor. The script checks whether the specified project already exists in Harbor.
If it does not exist, it proceeds to create one.

Usage:

$0 -a <harbor-rest-api-base-url> -u <username> -p <password> -m <project-name> -l <is-project-public>

Options:
 -a <harbor-rest-api-base-url>  Base URL of the Harbor REST API (Example https://<your-harbor-instance-domain>/api/v2.0)
 -u <username>                  Username with permissions to create a project in Harbor
 -p <password>                  Password for the corresponding username
 -m <project-name>              The name of the project to be created
 -l <is-project-public>         Access level of the harbor project (\"true\" for public project; \"false\" for private project)
 -h                             Display help usage
"""
exit 0
}

function create_project() {

  local fullProjectUrlExists="$REST_API_BASE_URL/projects?project_name=$PROJECT_NAME"

  # Check whether the project exists in Harbor
  echo "Check whether the project $PROJECT_NAME exists in Harbor: $fullProjectUrlExists"

  response=$(curl --user $USERNAME:$PASSWORD -I $fullProjectUrlExists -H "accept: application/json" --silent --output /dev/null -w "%{http_code}")

  # if the curl command succeeded
  if [ "$response" -eq 404 ]; then
    echo "Harbor project $PROJECT_NAME does not exist. Proceeding to create it."
    payload="{\"project_name\":\"$PROJECT_NAME\",\"metadata\":{\"public\":\"$IS_PUBLIC\"}}"
    response=$(curl --user $USERNAME:$PASSWORD -X POST $REST_API_BASE_URL/projects -H "accept: application/json" \
                -H "X-Resource-Name-In-Location: false" -H "Content-Type: application/json" \
                --silent --output /dev/null -w "%{http_code}" -d "$payload")
    if [ "$response" -eq 201 ]; then
      echo "The project $PROJECT_NAME was successfully created in Harbor"
    else
      echo "The project $PROJECT_NAME could not be successfully created in Harbor"
      return 1
    fi
  elif [ "$response" -eq 200 ]; then
    echo "Harbor project $PROJECT_NAME already exists."
  else
    echo "ERROR: curl call failed with response code: " $response
    return 1
  fi
}

exit_error() {
  usage
  exit 1
}

while getopts 'hu:p:a:m:l:' opt; do
  case $opt in
  a)
    REST_API_BASE_URL=$OPTARG
    ;;
  u)
    USERNAME=$OPTARG
    ;;
  p)
    PASSWORD=$OPTARG
    ;;
  m)
    PROJECT_NAME=$OPTARG
    ;;
  l)
    IS_PUBLIC=$OPTARG
    if [[ $IS_PUBLIC != "true" && $IS_PUBLIC != "false" ]]; then
      exit_error
    fi
    ;;
  h | ?)
    usage
    ;;
  esac
done

create_project
