#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

function usage {
    echo
    echo "usage: $0 [OPTIONS]"
    echo "  -c compartment_ocid       Compartment OCID of the DNS Zone."
    echo "  -e env_name               Environment name used by the Verrazzano resource. Defaults to 'default'."
    echo "  -z zone_name              Name of the DNS Zone."
    echo "  -s dns_scope              DNS Zone scope - 'GLOBAL' or 'PRIVATE'."
    echo "  -m mgmt_ip                IP of the Verrazznao management load balancer."
    echo "  -a app_ip                 IP of the Verrazznao application load balancer."
    echo "  -h                        Display this help message."
    echo
    exit 1
}

function log () {
    echo "$(date '+[%Y-%m-%d %I:%M:%S %p]') : $1"
}

function addDNSRecord() {
    oci dns record domain patch --domain "$1" --zone-name-or-id "$ZONE_NAME" --scope "$DNS_SCOPE" --items "[{\"domain\": \"$1\", \"isProtected\": true, \"rtype\": \"$2\", \"rdata\": \"$3\", \"ttl\": 300}]"
}

function createVerrazzanoDNSRecords() {
    log "Creating the 'A' records"
    addDNSRecord $DOMAIN_APP "A" $LB_APP_IP
    addDNSRecord $DOMAIN_MGMT "A" $LB_MGMT_IP
    log "Creating the 'CNAME' records"
    addDNSRecord "verrazzano.$ENV_NAME.$ZONE_NAME" "CNAME" $DOMAIN_MGMT
    addDNSRecord "keycloak.$ENV_NAME.$ZONE_NAME" "CNAME" $DOMAIN_MGMT
    addDNSRecord "rancher.$ENV_NAME.$ZONE_NAME" "CNAME" $DOMAIN_MGMT
    addDNSRecord "grafana.vmi.system.$ENV_NAME.$ZONE_NAME" "CNAME" $DOMAIN_MGMT
    addDNSRecord "prometheus.vmi.system.$ENV_NAME.$ZONE_NAME" "CNAME" $DOMAIN_MGMT
    addDNSRecord "kiali.vmi.system.$ENV_NAME.$ZONE_NAME" "CNAME" $DOMAIN_MGMT
    addDNSRecord "kibana.vmi.system.$ENV_NAME.$ZONE_NAME" "CNAME" $DOMAIN_MGMT
    addDNSRecord "elasticsearch.vmi.system.$ENV_NAME.$ZONE_NAME" "CNAME" $DOMAIN_MGMT
}

COMPARTMENT_OCID=""
ENV_NAME="default"
ZONE_NAME=""
DNS_SCOPE=""
LB_MGMT_IP=""
LB_APP_IP=""

while getopts c:e:z:s:m:a:h flag
do
    case "$flag" in
        c) COMPARTMENT_OCID=$OPTARG;;
        e) ENV_NAME=$OPTARG;;
        z) ZONE_NAME=$OPTARG;;
        s) DNS_SCOPE=$OPTARG;;
        m) LB_MGMT_IP=$OPTARG;;
        a) LB_APP_IP=$OPTARG;;
        h) usage;;
        *) usage;;
    esac
done

if [ -z "$COMPARTMENT_OCID" ] ; then
    log "Compartment OCID must be specified."
    exit 1
fi
if [ -z "$ENV_NAME" ] ; then
    log "Verrazzano environment name must be specified."
    exit 1
fi
if [ -z "$ZONE_NAME" ] ; then
    log "DNS Zone name must be specified."
    exit 1
fi
if [ -z "$DNS_SCOPE" ] ; then
    log "DNS Zone scope must be specified."
    exit 1
fi
if [ -z "$LB_MGMT_IP" ] ; then
    log "Management load balancer IP must be specified."
    exit 1
fi
if [ -z "$LB_APP_IP" ] ; then
    log "Application load balancer IP must be specified."
    exit 1
fi

set -o pipefail
DOMAIN_MGMT="ingress-mgmt.$ENV_NAME.$ZONE_NAME"
DOMAIN_APP="ingress-verrazzano.$ENV_NAME.$ZONE_NAME"
createVerrazzanoDNSRecords



