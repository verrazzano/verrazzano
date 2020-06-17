#!/usr/bin/env bash

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
. $SCRIPT_DIR/common.sh

set -u

KEYCLOAK_NS=keycloak
KEYCLOAK_CHART_VERSION=8.2.2
KEYCLOAK_IMAGE_TAG=10.0.1
ADMIN_USERNAME=admin
ADMIN_PASSWORD=$(openssl rand -base64 30 | tr -d "=+/" | cut -c1-10)
MYSQL_IMAGE_TAG=8.0.20
MYSQL_ROOT_PASSWORD=$(openssl rand -base64 30 | tr -d "=+/" | cut -c1-10)
MYSQL_USERNAME=keycloak
MYSQL_PASSWORD=${MYSQL_ROOT_PASSWORD}
VERRAZZANO_NS=verrazzano-system
VZ_SYS_REALM=verrazzano-system
VZ_USERNAME=verrazzano
VZ_PASSWORD=verrazzan0
DNS_PREFIX="verrazzano-ingress"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

if ! kubectl get secret --namespace ${VERRAZZANO_NS} ${VZ_USERNAME} ; then
  consoleerr "ERROR: Must run 3-install-verrazzano.sh and then rerun this script."
  exit 1
fi
VZ_NEW_PASSWORD=$(kubectl get secret --namespace ${VERRAZZANO_NS} ${VZ_USERNAME} -o jsonpath="{.data.password}" | base64 --decode; echo)

function set_INGRESS_IP() {
  if [ ${CLUSTER_TYPE} == "OKE" ]; then
    INGRESS_IP=$(kubectl get svc ingress-controller-nginx-ingress-controller -n ingress-nginx -o json | jq -r '.status.loadBalancer.ingress[0].ip')
  elif [ ${CLUSTER_TYPE} == "KIND" ]; then
    INGRESS_IP=$(kubectl get node ${KIND_CLUSTER_NAME}-control-plane -o json | jq -r '.status.addresses[] | select (.type == "InternalIP") | .address')
  fi
}

function cleanup_all {
  set +e
  helm uninstall keycloak --namespace ${KEYCLOAK_NS} > /dev/null 2>&1
  helm uninstall mysql --namespace ${KEYCLOAK_NS} > /dev/null 2>&1
  kubectl delete --all pvc --namespace ${KEYCLOAK_NS} > /dev/null 2>&1
  set -e
}

function install_mysql {
  # Create keycloak namespace if it does not exist
  if ! kubectl get namespace ${KEYCLOAK_NS} 2> /dev/null ; then
    kubectl create namespace ${KEYCLOAK_NS}
  fi

  # generate mysql-values.yaml file 
  cat <<EOF > ${TMP_DIR}/mysql-values.yaml
imageTag: ${MYSQL_IMAGE_TAG}
busybox:
    image: container-registry.oracle.com/os/oraclelinux
    tag: 7-slim

mysqlRootPassword: ${MYSQL_ROOT_PASSWORD}

mysqlUser: ${MYSQL_USERNAME}
mysqlPassword: ${MYSQL_PASSWORD}

mysqlDatabase: keycloak

initializationFiles:
  first-db.sql: |-
    CREATE DATABASE IF NOT EXISTS keycloak DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;
    USE keycloak;
    GRANT ALL ON keycloak.* TO '${MYSQL_USERNAME}'@'%';
    FLUSH PRIVILEGES;

imagePullPolicy: IfNotPresent

ssl:
  enabled: false
EOF

  # Install mysql helm chart
  helm upgrade mysql stable/mysql \
      --install \
      --namespace ${KEYCLOAK_NS} \
      -f ${TMP_DIR}/mysql-values.yaml
       
  # Wait for mysql pods to be up and running
  kubectl wait pods -l app=mysql -n ${KEYCLOAK_NS} --for=condition=Ready --timeout=300s
}

function install_keycloak {
  # Replace strings in keycloak.json file
  sed "s|ENV_NAME|${ENV_NAME}|g;s|DNS_SUFFIX|${DNS_SUFFIX}|g;s|VZ_SYS_REALM|${VZ_SYS_REALM}|g;s|VZ_USERNAME|${VZ_USERNAME}|g" $SCRIPT_DIR/config/keycloak.json > ${TMP_DIR}/keycloak-sed.json

  set +e
  if ! kubectl get secret ${KEYCLOAK_NS} keycloak-realm-cacert 2> /dev/null ; then
      kubectl delete secret keycloak-realm-cacert -n ${KEYCLOAK_NS}
  fi
  set -e

  # Create keycloak secret
  kubectl create secret generic keycloak-realm-cacert \
      -n ${KEYCLOAK_NS} \
      --from-file=realm.json=${TMP_DIR}/keycloak-sed.json \
      --from-file=public.crt=$SCRIPT_DIR/config/keycloak-public.crt

  # Add keycloak helm repo
  helm repo add codecentric https://codecentric.github.io/helm-charts
  
  # generate keycloak-values.yaml file 
  cat <<EOF > ${TMP_DIR}/keycloak-values-sed.yaml
keycloak:
  extraInitContainers: |
    - name: theme-provider
      image: phx.ocir.io/stevengreenberginc/verrazzano/keycloak-oracle-theme:0.1
      imagePullPolicy: IfNotPresent
      command:
        - sh
      args:
        - -c
        - |
          echo "Copying theme..."
          cp -R /oracle/* /theme
      volumeMounts:
        - name: theme
          mountPath: /theme
        - name: cacerts
          mountPath: /cacerts          

  replicas: 1
  image:
    tag: ${KEYCLOAK_IMAGE_TAG}
  extraArgs: -Dkeycloak.import=/etc/keycloak/realm.json
  ## Username for the initial Keycloak admin user
  username: ${ADMIN_USERNAME}
  password: ${ADMIN_PASSWORD}

  containerSecurityContext:
    runAsUser: 0
    runAsNonRoot: false
  
  extraVolumes: |
    - name: keycloak-config
      secret:
        secretName: keycloak-realm-cacert
    - name: theme
      emptyDir: {} 
    - name: cacerts
      emptyDir: {}
  extraVolumeMounts: |
    - name: keycloak-config
      mountPath: /etc/keycloak
    - name: theme
      mountPath: /opt/jboss/keycloak/themes/oracle
  service:
    port: 8083
  ingress:
    enabled: true
    path: /

    annotations:
      external-dns.alpha.kubernetes.io/target: "${DNS_TARGET_NAME}"
      kubernetes.io/ingress.class: nginx
      kubernetes.io/tls-acme: "true"
      external-dns.alpha.kubernetes.io/ttl: "60"

    ## List of hosts for the ingress
    hosts:
      - keycloak.${ENV_NAME}.${DNS_SUFFIX}

    tls:
      - hosts:
          - keycloak.${ENV_NAME}.${DNS_SUFFIX}
        secretName: ${ENV_NAME}-secret

  persistence:
    deployPostgres: false
    dbVendor: mysql
    dbPassword: ${MYSQL_PASSWORD}
    dbUser: ${MYSQL_USERNAME}
    dbHost: mysql
    dbPort: 3306
EOF
 
  # Install keycloak helm chart
  helm upgrade keycloak codecentric/keycloak \
      --install \
      --namespace ${KEYCLOAK_NS} \
      --version ${KEYCLOAK_CHART_VERSION} \
      -f ${TMP_DIR}/keycloak-values-sed.yaml

  # Wait for keycloak to be up and running
  kubectl wait pod/keycloak-0 -n ${KEYCLOAK_NS} --for=condition=Ready --timeout=300s

  # Update to use Oracle login theme settings for keycloak
  kubectl exec keycloak-0 \
      -n ${KEYCLOAK_NS} \
      -c keycloak \
      -- opt/jboss/keycloak/bin/kcadm.sh update realms/master -s loginTheme=oracle --no-config --server http://localhost:8080/auth --realm master --user ${ADMIN_USERNAME} --password ${ADMIN_PASSWORD}

  # Reset verrazzano-system/verrazzano password
  kubectl exec keycloak-0 \
          -n ${KEYCLOAK_NS} \
          -c keycloak \
          -- opt/jboss/keycloak/bin/kcadm.sh update users/f37bf86b-7f56-4f39-b71d-953078fbb870/reset-password --server http://localhost:8080/auth --realm ${VZ_SYS_REALM} --user ${VZ_USERNAME} --password ${VZ_PASSWORD} -s type=password -s value=${VZ_NEW_PASSWORD} -n

  # Wait for TLS cert from Cert Manager to go into a ready state
  kubectl wait cert/${ENV_NAME}-secret -n keycloak --for=condition=Ready
}

function usage {
    consoleerr
    consoleerr "usage: $0 [-n name] [-d dns_type] [-s dns_suffix]"
    consoleerr "  -n name        Environment Name. Optional.  Optional.  Defaults to default."
    consoleerr "  -d dns_type    DNS type [xip.io|oci]. Optional.  Defaults to xip.io."
    consoleerr "  -s dns_suffix  DNS suffix (e.g v8o.oracledx.com). Not valid for dns_type xip.io. Required for dns-type oci."
    consoleerr "  -h             Help"
    consoleerr
    exit 1
}

ENV_NAME="default"
DNS_TYPE="xip.io"
DNS_SUFFIX=""

while getopts n:d:s:h flag
do
    case "${flag}" in
        n) ENV_NAME=${OPTARG};;
        d) DNS_TYPE=${OPTARG};;
        s) DNS_SUFFIX=${OPTARG};;
        h) usage;;
    esac
done
# check for valid DNS type
if [ $DNS_TYPE != "xip.io" ] && [ $DNS_TYPE != "oci" ]; then
  consoleerr
  consoleerr "Unknown DNS type ${DNS_TYPE}"
  usage
fi
# check for name
if [ $DNS_TYPE = "oci" ]; then
  if [ -z "$ENV_NAME" ]; then
    consoleerr
    consoleerr "Name must be given with dns_type oci!"
    usage
  fi
fi

if [ $DNS_TYPE = "xip.io" ]; then
  set_INGRESS_IP
fi

# check expected dns suffix for given dns type
if [ -z "$DNS_SUFFIX" ]; then
  if [ $DNS_TYPE = "oci" ]; then
    consoleerr
    consoleerr "-s option is required for ${DNS_TYPE}"
    usage
  else
    DNS_SUFFIX="${INGRESS_IP}".xip.io
  fi
else
  if [ $DNS_TYPE = "xip.io" ]; then
    consoleerr
    consoleerr "A dns_suffix should not be given with dns_type xip.io!"
    usage
  fi
fi

DNS_TARGET_NAME=${DNS_PREFIX}.${ENV_NAME}.${DNS_SUFFIX}

action "Cleaning up previous installation" cleanup_all || exit 1
action "Installing MySQL" install_mysql || exit 1
action "Installing Keycloak" install_keycloak || exit 1

rm -rf $TMP_DIR

consoleout
consoleout "To retrieve the initial keycloak administrator ${ADMIN_USERNAME} password run:"
consoleout "kubectl get secret --namespace keycloak keycloak-http -o jsonpath="{.data.password}" | base64 --decode; echo"

