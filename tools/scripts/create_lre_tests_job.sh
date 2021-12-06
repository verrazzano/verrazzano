#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# This script can be used to execute compiled test binaries as a kubernetes job in a cluster. 
# The target cluster where the tests will run can be different than where the job is being executed.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

function usage {
    echo
    echo "Usage:"
    echo " $0 [-f TEST_BINARIES] [-i JOB_IMAGE]"
    echo "Arguments:"
    echo "  -i, --job-image               The docker image of of the job."
    echo "  -j, --job-name                (Optional) The name of of the job."
    echo "  -n, --job-namespace           (Optional) The namespace of of the job."
    echo "  -s, --source-kubeconfig       (Optional) Path to Kubeconfig of cluster where job will run. If not provided, env variable KUBECONFIG is used."
    echo "  -t, --target-kubeconfig       (Optional) Path to Kubeconfig of cluster where tests wll be run. If not provided, --source-kubeconfig is used."
    echo "  -b, --oci-bucket              (Optional) Test log bucket. Required for pushing logs to object storage. Please make sure OCI CLI env variables are set."
    echo "  -p, --prometheus-pushgateway  (Optional) URL to prometheus pushgateway for pushing metrics. It MUST not have a trailing slash ('/')." 
    echo "  -o, --prometheus-job          (Optional) Job name for prometheus metrics."
    echo "  -m, --prometheus-instace      (Optional) Instace name for prometheus metrics."
    echo "  -u, --sauron-cred             (Optional) Sauron credentials in the format: <username>:<password>"
    echo "  -r, --regex-filter            (Optional) The regex filter for tests used by Ginkgo."
    echo "  -l, --test-log-archive        (Optional) The name of test log archive file."
    echo "  -d, --sleep-duration          (Optional). Time to sleep after every iteration, default is 1 minute."
    echo "  -v, --verbose                 (Optional) Turn on verbose output."
    echo "  -h, --help                    Print usage information."
    echo
    exit 1
}

SOURCE_KUBECONFIG=""
TARGET_KUBECONFIG=""
JOB_NAME=""
JOB_NAMESPACE=""
JOB_IMAGE=""
TEST_BINARIES="/tmp/verrazzano/test-binaries"
PROMETHEUS_GW_URL=""
PROMETHEUS_JOB=""
PROMETHEUS_INSTANCE=""
SAURON_CRED=""
OCI_BUCKET=""
TEST_REGEX=""
TEST_LOG_ARCHIVE=""
SLEEP_DURATION=""
VERBOSE=""
TS=`date "+%Y%m%d-%H%M%S%s"`
JOB_NAME_DEFAULT="testjob-${TS}"
JOB_NAMESPACE_DEFAULT="lre-tests"

# Parse the command line arguments
while [ $# -gt 0 ]
do
    key="$1"
    case ${key} in 
        -s|--source-kubeconfig)        SOURCE_KUBECONFIG="$2"; shift; shift;;
        -t|--target-kubeconfig)        TARGET_KUBECONFIG="$2"; shift; shift;;
        -j|--job-name)                 JOB_NAME="$2"; shift; shift;;
        -n|--job-namespace)            JOB_NAMESPACE="$2"; shift; shift;;
        -i|--job-image)                JOB_IMAGE="$2"; shift; shift;;
        -b|--oci-bucket)               OCI_BUCKET="$2"; shift; shift;;
        -p|--prometheus-pushgateway)   PROMETHEUS_GW_URL="$2"; shift; shift;;
        -o|--prometheus-job)           PROMETHEUS_JOB="$2"; shift; shift;;
        -m|--prometheus-instance)      PROMETHEUS_INSTANCE="$2"; shift; shift;;
        -u|--sauron-cred)              SAURON_CRED="$2"; shift; shift;;
        -r|--regex-filter)             TEST_REGEX="$2"; shift; shift;;
        -l|--test-log-archive)         TEST_LOG_ARCHIVE="$2"; shift; shift;;
        -d|--sleep-duration)           SLEEP_DURATION="$2"; shift; shift;;
        -v|--verbose)                  VERBOSE=true; shift;;
        -h|--help)                     usage;;
	*)                             echo "ERROR: Invalid argument: ${key}"; usage;;
    esac
done

SOURCE_KUBECONFIG=${SOURCE_KUBECONFIG:-$KUBECONFIG}
if [ -z "${SOURCE_KUBECONFIG}" ]
then
    echo "ERROR: Either --source-kubeconfig should be provided or env variable KUBECONFIG should be set."
    exit 1
elif [ -z "${JOB_IMAGE}" ]
then
    echo "ERROR: Missing required argument: -i, --job-image"
    usage
fi

# Create namespace
if [ -z "${TARGET_KUBECONFIG}" ]
then
    TARGET_KUBECONFIG=${SOURCE_KUBECONFIG}
fi
if [ -z "${JOB_NAME}" ]
then
    JOB_NAME=${JOB_NAME_DEFAULT}
fi
if [ -z "${JOB_NAMESPACE}" ]
then
    JOB_NAMESPACE=${JOB_NAMESPACE_DEFAULT}
fi
export KUBECONFIG=${SOURCE_KUBECONFIG}
kubectl create namespace ${JOB_NAMESPACE}

# Create configmap
mkdir -p /tmp/${JOB_NAME}
cp ${TARGET_KUBECONFIG} /tmp/${JOB_NAME}/kubeconfig
cp ${SCRIPT_DIR}/run_compiled_tests.sh /tmp/${JOB_NAME}
CONFIGMAP_NAME="${JOB_NAME}-configmap"
kubectl create configmap ${CONFIGMAP_NAME} -n ${JOB_NAMESPACE} --from-file=/tmp/${JOB_NAME}

# Command to execute in the pod
CMDLINE="sleep 5s && \
        cd /tmp && \
        git clone https://github.com/verrazzano/verrazzano && \
        mkdir -p ${TEST_BINARIES} && \
        ./lre-config/run_compiled_tests.sh --test-binaries ${TEST_BINARIES} --kubeconfig-location /tmp/lre-config/kubeconfig"

# Add arguments for the script - run_compiled_tests.sh
if [ ! -z "${PROMETHEUS_GW_URL}" ]
then
    CMDLINE="${CMDLINE} --prometheus-pushgateway ${PROMETHEUS_GW_URL}"
fi
if [ ! -z "${PROMETHEUS_JOB}" ]
then
    CMDLINE="${CMDLINE} --prometheus-job ${PROMETHEUS_JOB}"
fi
if [ ! -z "${PROMETHEUS_INSTANCE}" ]
then
    CMDLINE="${CMDLINE} --prometheus-instance ${PROMETHEUS_INSTANCE}"
fi
if [ ! -z "${SAURON_CRED}" ]
then
    CMDLINE="${CMDLINE} --sauron-cred ${SAURON_CRED}"
fi
if [ ! -z "${TEST_REGEX}" ]
then
    CMDLINE="${CMDLINE} --test-regex ${TEST_REGEX}"
fi
if [ ! -z "${TEST_LOG_ARCHIVE}" ]
then
    CMDLINE="${CMDLINE} --test-log-archive ${TEST_LOG_ARCHIVE}"
fi
if [ ! -z "${SLEEP_DURATION}" ]
then
    CMDLINE="${CMDLINE} --sleep-duration ${SLEEP_DURATION}"
fi
if [ ! -z "${VERBOSE}" ]
then
    CMDLINE="${CMDLINE} --verbose"
fi

# Create OCI secret
SECRET_NAME="oci"
if [ ! -z "${OCI_BUCKET}" ]
then
    kubectl get secret ${SECRET_NAME} -n ${JOB_NAMESPACE} > /dev/null 2>&1
    if [ $? -ne 0 ]
    then
      ${SCRIPT_DIR}/../../platform-operator/scripts/install/create_oci_config_secret.sh ${JOB_NAMESPACE}
    fi
    CMDLINE="${CMDLINE} --oci-bucket ${OCI_BUCKET}"
fi

kubectl apply -f - <<-EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: ${JOB_NAMESPACE}
spec:
  template:
    metadata:
      name: ${JOB_NAME}
    spec:
      containers:
        - name: ${JOB_NAME}
          image: ${JOB_IMAGE}
          env:
             - name: KUBECONFIG
               value: /tmp/lre-config/kubeconfig
          volumeMounts:
            - name: ocisecret
              mountPath: /var/run/secrets/ocisecret
            - name: lre-config
              mountPath: /tmp/lre-config
          command: [ "/bin/sh", "-c"]
          args: [${CMDLINE}]
      restartPolicy: Never
      volumes:
      - name: ocisecret
        secret:
          secretName: ${SECRET_NAME}
          optional: true
      - name: lre-config
        projected:
          sources:
            - configMap:
                name: ${CONFIGMAP_NAME}
                items:
                  - key: kubeconfig
                    path: kubeconfig
                    mode: 0755
                  - key: run_compiled_tests.sh
                    path: run_compiled_tests.sh
                    mode: 0755
EOF

kubectl wait -n ${JOB_NAMESPACE} --for=condition=ContainersReady --timeout=400s pod --selector job-name=${JOB_NAME}

POD_NAME=$(kubectl get pod -n ${JOB_NAMESPACE} -l job-name=${JOB_NAME} -o jsonpath="{.items[0].metadata.name}") 
echo
echo "Copy test binaries using the following command:"
echo "kubectl cp -n ${JOB_NAMESPACE} <path-to-binaries-folder> ${POD_NAME}:${TEST_BINARIES}"
echo
kubectl logs -n ${JOB_NAMESPACE} ${POD_NAME} -f
