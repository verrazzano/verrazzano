#!/usr/bin/env bash

oci --region us-ashburn-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/image-list --file ${WORKSPACE}/image-sizes.txt
if [ $? -ne  0 ] ; then
  echo "image-sizes.txt not found"
  oci --region us-ashburn-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${CLEAN_BRANCH_NAME}/image-list --file ${WORKSPACE}/image-sizes.txt
  if [ $? -ne 0 ] ; then
     echo "os object put --file image-sizes.txt failed"
     exit
  fi
fi

diff $IMAGE_LIST 1x.txt > out.txt }
if [ $? -eq 0 ] ; then
  rm out.txt
fi
