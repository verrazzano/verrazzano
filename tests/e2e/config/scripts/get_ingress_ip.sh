
TEST_CONFIG_FILE=$1
INGRESS_IP=$(kubectl get svc ingress-controller-ingress-nginx-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
sed -i "s/XX_DNS_ZONE_XX/${INGRESS_IP}.xip.io/" ${TEST_CONFIG_FILE}
