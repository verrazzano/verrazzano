#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage {
    echo
    echo "usage: $0 -c compartment_ocid -s subnet_ocid -n lb_name -b backend_ip [OPTIONS]"
    echo "  -o operation              Operation to perform - 'create' or 'delete'."
    echo "  -c compartment_ocid       Compartment OCID for creating the load balancer."
    echo "  -n lb_name                Display name for the load balancer."
    echo "  -s subnet_ocid            Subnet OCID for creating the load balancer. Required by 'create' operation."
    echo "  -b backend_ip             Space separated string of backend IPs. Required by 'create' operation."
    echo "                            Example - \"1.1.1.1 2.2.2.2 3.3.3.3\""
    echo "  -p backend_ports          Space separated string of backend ports ('http' followed by 'https'). Required by 'create' operation."
    echo "                            Example - \"30080 30443\""
    echo "  -f template_file_path     Path to the json template file for creating load balancer. Used by 'create' operation."
    echo "  -l lb_shape               Shape for the load balancer to be created. Defaults to '10Mbps'. Used by 'create' operation."
    echo "                            To check the available shapes in the compartment, run the following command -"
    echo "                            oci lb shapes list --compartment-id <COMPARTMENT_OCID>"
    echo "  -e enable_public_ip       Provide a public IP to the load balancer. Defaults to 'false'. Used by 'create' operation." 
    echo "                            The subnet specififed for the load balancer MUST be a public subnet."
    echo "  -h                        Display this help message."
    echo
    exit 1
}

OPERATION=""
COMPARTMENT_OCID=""
SUBNET_OCID=""
LB_NAME=""
LB_SHAPE="10Mbps"
BACKEND_IP=""
BACKEND_PORTS=""
TEMPLATE_FILE_PATH=""
ENABLE_PUBLIC_IP="false"

log () {
    echo "$(date '+[%Y-%m-%d %I:%M:%S %p]') : $1"
}

function createLoadBalancer() {
    log "Creating a load balancer: $LB_NAME with shape: $LB_SHAPE"
    cp $TEMPLATE_FILE_PATH "lb.json"
    jq --arg compartment_ocid "$COMPARTMENT_OCID" '.compartmentId = $compartment_ocid' lb.json > "tmp" && mv "tmp" lb.json
    jq --arg lb_name "$LB_NAME" '.displayName = $lb_name' lb.json > "tmp" && mv "tmp" lb.json
    jq --arg lb_shape "$LB_SHAPE" '.shapeName = $lb_shape' lb.json > "tmp" && mv "tmp" lb.json
    jq --arg subnet_ocid "$SUBNET_OCID" '.subnetIds[0] = $subnet_ocid' lb.json > "tmp" && mv "tmp" lb.json
    read -a ports <<< $BACKEND_PORTS
    http_port=${ports[0]}
    https_port=${ports[1]}
    for ip in ${BACKEND_IP}; do
      jq --argjson http_port $http_port --arg ip "$ip" '.backendSets.http.backends += [{"ipAddress": $ip, "port": $http_port, "weight": 1}]' lb.json > "tmp" && mv "tmp" lb.json
      jq --argjson https_port $https_port --arg ip "$ip" '.backendSets.https.backends += [{"ipAddress": $ip, "port": $https_port, "weight": 1}]' lb.json > "tmp" && mv "tmp" lb.json
    done
    jq --argjson http_port $http_port '.backendSets.http.healthChecker.port = $http_port' lb.json > "tmp" && mv "tmp" lb.json
    jq --argjson https_port $https_port '.backendSets.https.healthChecker.port = $https_port' lb.json > "tmp" && mv "tmp" lb.json
    if [ "$ENABLE_PUBLIC_IP" == "true" ]; then
      jq '.isPrivate = "false"' lb.json > "tmp" && mv "tmp" lb.json
    fi
    oci lb load-balancer create --from-json file://lb.json
    rm lb.json
    if [ $? -ne 0 ]; then 
        log "Failed to create the load balancer: $LB_NAME"
        exit 1
    fi
    LB_OCID=$(oci lb load-balancer list --compartment-id $COMPARTMENT_OCID --display-name $LB_NAME | jq -r '.data[0].id')
    log "Successfully created the load balancer: $LB_NAME"
    log "Load balancer OCID: $LB_OCID"
}

function deleteLoadBalancer() {
    log "Deleting the load balancer: $LB_NAME"
    LB_OCID=$(oci lb load-balancer list --compartment-id $COMPARTMENT_OCID --display-name $LB_NAME | jq -r '.data[0].id')
    if [ $? -ne 0 ]; then
        log "Error while fetching load balancer: $LB_NAME."
        exit 1
    fi
    log "Load balancer OCID: $LB_OCID"
    oci lb load-balancer delete --load-balancer-id $LB_OCID --force --wait-for-state SUCCEEDED
    if [ $? -ne 0 ]; then
        log "Error while deleting load balancer: $LB_NAME."
        exit 1
    fi
    log "Successfully deleted the load balancer: $LB_NAME"
}

while getopts o:c:s:n:l:b:p:f:e:h flag
do
    case "$flag" in
        o) OPERATION=$OPTARG;;
        c) COMPARTMENT_OCID=$OPTARG;;
        s) SUBNET_OCID=$OPTARG;;
        n) LB_NAME=$OPTARG;;
        l) LB_SHAPE=$OPTARG;;
        b) BACKEND_IP=$OPTARG;;
        p) BACKEND_PORTS=$OPTARG;;
        f) TEMPLATE_FILE_PATH=$OPTARG;;
        e) ENABLE_PUBLIC_IP=$OPTARG;;
        h) usage;;
        *) usage;;
    esac
done

if [ -z "$OPERATION" ] ; then
    log "Operation must be specified."
    exit 1
fi
if [ -z "$COMPARTMENT_OCID" ] ; then
    log "Compartment OCID must be specified."
    exit 1
fi
if [ -z "$LB_NAME" ] ; then
    log "Load balancer name must be specified."
    exit 1
fi
if [ $OPERATION == "create" ]; then
    if [ -z "$SUBNET_OCID" ] ; then
        log "Subnet OCID must be specified."
        exit 1
    fi
    if [ -z "$BACKEND_IP" ] ; then
        log "Backend IPs must be specified."
        exit 1
    fi
    if [ -z "$BACKEND_PORTS" ] ; then
        log "Backend ports must be specified."
        exit 1
    fi
    if [ -z "$TEMPLATE_FILE_PATH" ] ; then
        log "Path to the json template file must be specified."
        exit 1
    fi
fi

set -o pipefail
if [ $OPERATION == "create" ]; then
    createLoadBalancer
elif [ $OPERATION == "delete" ]; then
    deleteLoadBalancer
else
    log "Invalid operation value: $OPERATION."
    usage
    exit 1
fi
