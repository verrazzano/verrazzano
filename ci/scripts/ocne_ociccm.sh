#!/bin/bash

#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

get_security_list_id() {
    n=0
    while [ $n -le 30 ] && [ -z "${id}" ]; do
        id=$(oci network security-list list --display-name "${TF_VAR_prefix}-lb-subnet" --compartment-id "${TF_VAR_compartment_id}" --vcn-id "${VCN_OCID}" --query 'data[0]."id"' --raw-output)
        n=$((n+1))
        sleep 2
    done
    echo "${id}"
}

get_subnet_id() {
    n=0
    while [ $n -le 30 ] && [ -z "${id}" ]; do
        id=$(oci network subnet list --display-name "${TF_VAR_prefix}-lb-subnet" --compartment-id "${TF_VAR_compartment_id}" --vcn-id "${VCN_OCID}" --query 'data[0]."id"' --raw-output)
        n=$((n+1))
        sleep 2
    done
    echo "${id}"
}

get_route_table_id() {
    n=0
    while [ $n -le 30 ] && [ -z "${id}" ]; do
        id=$(oci network route-table list --display-name internet-route --compartment-id "${TF_VAR_compartment_id}" --vcn-id "${VCN_OCID}" --query 'data[0]."id"' --raw-output)
        n=$((n+1))
        sleep 2
    done
    echo "${id}"
}

get_dhcp_options_id() {
    n=0
    while [ $n -le 30 ] && [ -z "${id}" ]; do
        id=$(oci network dhcp-options list --display-name ocne-dhcp-options --compartment-id "${TF_VAR_compartment_id}" --vcn-id "${VCN_OCID}" --query 'data[0]."id"' --raw-output)
        n=$((n+1))
        sleep 2
    done
    echo "${id}"
}

create_security_list() {
    lb_subnet_ingress=$(mktemp)
    lb_subnet_egress=$(mktemp)
    trap 'rm -f ${lb_subnet_ingress} ${lb_subnet_egress}' EXIT

    cat > "${lb_subnet_ingress}" << EOL
[
    {
        "protocol": "6",
        "source": "0.0.0.0/0",
        "tcpOptions": {
            "destinationPortRange": {
                "max": 443,
                "min": 443
            }
        }
    }
]
EOL

    cat > "${lb_subnet_egress}" << EOL
[
    {
        "destination": "0.0.0.0/0",
        "protocol": "all"
    }
]
EOL

    oci network security-list create --display-name "${TF_VAR_prefix}-lb-subnet" \
        --compartment-id "${TF_VAR_compartment_id}" --vcn-id "${VCN_OCID}" \
        --ingress-security-rules file://"${lb_subnet_ingress}" --egress-security-rules file://"${lb_subnet_egress}"

}

create_subnet() {
    igrt_id=$(get_route_table_id)
    dhcp_id=$(get_dhcp_options_id)
    oci network subnet create --display-name "${TF_VAR_prefix}-lb-subnet" \
        --compartment-id "${TF_VAR_compartment_id}" --vcn-id "${VCN_OCID}" \
        --cidr-block "10.0.2.0/24" --dns-label "lb" --security-list-ids "[\""${security_list_id}"\"]" \
        --prohibit-public-ip-on-vnic false --prohibit-internet-ingress false \
        --route-table-id "${igrt_id}" --dhcp-options-id "${dhcp_id}" 
}

deployCCM() {
    ociccm_name="ociccm"
    echo "deployCCM with lb_subnet_id ${lb_subnet_id}"
    scp -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" "${TF_VAR_api_private_key_path}" opc@"${API_SERVER_IP}":/home/opc/oci_api_deployer_key.pem
    ssh -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" opc@"${API_SERVER_IP}" -- olcnectl --api-server="${API_SERVER_IP}":8091 \
        module create -E "${OCNE_ENVNAME}" -M oci-ccm -N "${ociccm_name}" \
            --oci-private-key-file /home/opc/oci_api_deployer_key.pem \
            --oci-ccm-kubernetes-module ${OCNE_K8SNAME} \
            --oci-region  "${TF_VAR_region}" \
            --oci-tenancy "${TF_VAR_tenancy_id}" \
            --oci-compartment "${TF_VAR_compartment_id}" \
            --oci-user "${TF_VAR_user_id}" \
            --oci-fingerprint "${TF_VAR_fingerprint}" \
            --oci-vcn "${VCN_OCID}" \
            --oci-lb-subnet1 "${lb_subnet_id}" \
            --oci-lb-security-mode None
    ssh -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" opc@"${API_SERVER_IP}" -- olcnectl --api-server="${API_SERVER_IP}":8091 \
        module install -E "${OCNE_ENVNAME}" -N "${ociccm_name}"
    ssh -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" opc@"${API_SERVER_IP}" -- olcnectl --api-server="${API_SERVER_IP}":8091 \
        module instances -E "${OCNE_ENVNAME}"
}

deployCCM15() {
    helm_name="myhelm"
    ociccm_name="ociccm"
    cp_count="${TF_VAR_control_plane_node_count}"
    cp_nodes=$(terraform output -json control_plane_nodes)
    for (( i=0; i<$cp_count; i++ )) do
      control_plane_node=$(echo "${cp_nodes}" | jq -r ".[$i]")
      scp -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" "${TF_VAR_api_private_key_path}" opc@"${control_plane_node}":/home/opc/oci_api_deployer_key.pem
    done
    # install helm module
    ssh -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" opc@"${API_SERVER_IP}" -- olcnectl --api-server="${API_SERVER_IP}":8091 \
        module create -E ${OCNE_ENVNAME} -M helm -N ${helm_name} --helm-kubernetes-module ${OCNE_K8SNAME} 
    ssh -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" opc@"${API_SERVER_IP}" -- olcnectl --api-server="${API_SERVER_IP}":8091 \
        module install -E "${OCNE_ENVNAME}" -N ${helm_name}
    # install oci-ccm module
    echo "Install oci-ccm to ${API_SERVER_IP}"
    ssh -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" opc@"${API_SERVER_IP}" -- olcnectl --api-server="${API_SERVER_IP}":8091 \
        module create -E "${OCNE_ENVNAME}" -M oci-ccm -N "${ociccm_name}" \
            --oci-private-key /home/opc/oci_api_deployer_key.pem \
            --oci-ccm-helm-module ${helm_name} \
            --oci-region  "${TF_VAR_region}" \
            --oci-tenancy "${TF_VAR_tenancy_id}" \
            --oci-compartment "${TF_VAR_compartment_id}" \
            --oci-user "${TF_VAR_user_id}" \
            --oci-fingerprint "${TF_VAR_fingerprint}" \
            --oci-vcn "${VCN_OCID}" \
            --oci-lb-subnet1 "${lb_subnet_id}" \
            --oci-lb-security-mode None
    ssh -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" opc@"${API_SERVER_IP}" -- olcnectl --api-server="${API_SERVER_IP}":8091 \
        module install -E "${OCNE_ENVNAME}" -N "${ociccm_name}"
    ssh -o StrictHostKeyChecking=no -i "${TF_VAR_ssh_private_key_path}" opc@"${API_SERVER_IP}" -- olcnectl --api-server="${API_SERVER_IP}":8091 \
        module instances -E "${OCNE_ENVNAME}"
}

create_security_list
security_list_id=$(get_security_list_id)

if [ -z "${security_list_id}" ]; then
    echo "Failed to create security list"
    exit 1
else
    create_subnet
    lb_subnet_id=$(get_subnet_id)
    if [ -z "${lb_subnet_id}" ]; then
        echo "Failed to create subnet"
        exit 1
    else
        echo "Installing OCNE-${OCNE_VERSION} OCI-CCM module with LB subnet ${lb_subnet_id}"
        if [[ "${OCNE_VERSION}" == "1.5"* ]]; then
            deployCCM15
        else
            deployCCM
        fi
    fi
fi

