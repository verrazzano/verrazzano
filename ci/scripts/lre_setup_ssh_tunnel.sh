#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

if [ -z "${ssh_private_key_path}" ] ; then
    echo "ssh_private_key_path env var must be set!"
    exit 1
fi
if [ -z "${ssh_public_key_path}" ] ; then
    echo "ssh_public_key_path env var must be set!"
    exit 1
fi
if [ -z "${dev_lre_compartment_id}" ] ; then
    echo "dev_lre_compartment_id env var must be set!"
    exit 1
fi
if [ -z "${KUBECONFIG}" ] ; then
    echo "KUBECONFIG env var must be set!"
    exit 1
fi


BASTION_ID=$(oci bastion bastion list \
            --compartment-id "${dev_lre_compartment_id}" --all \
            | jq -r '.data[0]."id"')

if [ -z "$BASTION_ID" ]; then
    echo "Failed to get the BASTION_ID"
    exit 1
fi

SESSION_ID=$(oci bastion session create-port-forwarding \
   --bastion-id $BASTION_ID \
   --ssh-public-key-file ${ssh_public_key_path} \
   --target-private-ip 10.196.0.58 \
   --target-port 6443 | jq '.data.id' | sed s/\"//g)

if [ -z "$SESSION_ID" ]; then
    echo "Failed to create a bastion session"
    exit 1
fi

echo "Waiting for $SESSION_ID to start"
sleep 60

COMMAND=`oci bastion session get  --session-id=${SESSION_ID} | \
  jq '.data."ssh-metadata".command' | \
  sed 's/"//g' | \
  sed 's|ssh|ssh -t|g' | \
  sed 's|<privateKey>|'"${ssh_private_key_path}"'|g' | \
  sed 's|<localPort>|6443|g'`
echo "command = ${COMMAND}"
if [ -z "$COMMAND" ]; then
    echo "didn't find the command to set up ssh tunnel"
    exit 1
fi
COMMAND+=" -o StrictHostKeyChecking=no -v -4 'ssh -t -L 6443:10.244.1.78:8443 keycloakadmin@10.244.1.78' &"
echo "command = ${COMMAND}"
echo "Setting up the ssh tunnel"
eval ${COMMAND}


if [ $? -ne 0 ]; then
  echo "Failed to setup ssh tunnel to the bastion host ${BASTION_ID}"
  exit 1
fi

echo "Successfully set up ssh tunnel"