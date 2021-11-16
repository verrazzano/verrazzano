#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)
function usage {
    echo
    echo "usage: $0 -c source_cluster_kube_config_path -k target_cluster_kube_config_path -t test_binary_url_or_file -r regex_filter_for_tests -o test_logs_os_bucket -l test_log_archive_name -j job_name -n job_namespace -i job_image"
    echo "  -c source_cluster_kube_config_path        Path to KUBECONFIG of cluster where job will run"
    echo "  -k target_cluster_kube_config_path        Path to KUBECONFIG of cluster where tests wll be run"
    echo "  -t test_binary_url_or_file                The url or path of test binary file"
    echo "  -r regex_filter_for_tests                 The regex filter for tests"
    echo "  -o test_logs_os_bucket                    The OS Bucket name for test logs, please make sure OCI CLI env variables are set."
    echo "  -l test_log_archive_name                  The name of test log archive file"
    echo "  -j job_name                               The name of of the job"
    echo "  -n job_namespace                          The namespace of of the job"
    echo "  -i job_image                              The docker image of of the job"
    echo "  -h help                                   show this help"

    echo
    exit 1
}

SOURCE_KUBECONFIG=""
TARGET_KUBECONFIG=""
TEST_BINARY=""
TEST_REGEX=""
TEST_LOG_BUCKET=""
TEST_LOG_ARCHIVE=""
JOB_NAME=""
JOB_NAMESPACE=""
JOB_IMAGE=""
TS=`date "+%Y%m%d-%H%M%S%s"`
JOB_NAME_DEFAULT="testjob-${TS}"
JOB_NAMESPACE_DEFAULT=default

while getopts c:k:t:r:o:l:j:n:i:h flag
do
    case "${flag}" in
        c) SOURCE_KUBECONFIG=${OPTARG};;
        k) TARGET_KUBECONFIG=${OPTARG};;
        t) TEST_BINARY=${OPTARG};;
        r) TEST_REGEX=${OPTARG};;
        o) TEST_LOG_BUCKET=${OPTARG};;
        l) TEST_LOG_ARCHIVE=${OPTARG};;
        j) JOB_NAME=${OPTARG};;
        n) JOB_NAMESPACE=${OPTARG};;
        i) JOB_IMAGE=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

SOURCE_KUBECONFIG=${SOURCE_KUBECONFIG:-$KUBECONFIG}
if [ -z "${SOURCE_KUBECONFIG}" ] ; then
    echo "SOURCE_KUBECONFIG must be set!"
    exit 1
fi

if [ -z "${JOB_IMAGE}" ] ; then
    echo "JOB_IMAGE must be set!"
    exit 1
fi

if [ -z "${TEST_BINARY}" ] ; then
    echo "TEST_BINARY must be set!"
    exit 1
fi

if [ -z "${TARGET_KUBECONFIG}" ] ; then
    TARGET_KUBECONFIG=${SOURCE_KUBECONFIG}
fi

export KUBECONFIG=${SOURCE_KUBECONFIG}
if [ -z "${JOB_NAME}" ] ; then
    JOB_NAME=${JOB_NAME_DEFAULT}
fi

if [ -z "${JOB_NAMESPACE}" ] ; then
    JOB_NAMESPACE=${JOB_NAMESPACE_DEFAULT}
fi

if [ ! -z "${TEST_LOG_BUCKET}" ] ; then
    kubectl create ns ${JOB_NAMESPACE}
    kubectl get secret oci -n ${JOB_NAMESPACE} > /dev/null 2>&1
    if [ $? -ne 0 ]; then
      ${SCRIPT_DIR}/../../platform-operator/scripts/install/create_oci_config_secret.sh ${JOB_NAMESPACE}
    fi
fi

export CM_DATA_KUBECONFIG=`cat ${TARGET_KUBECONFIG}`
export CM_DATA_RUN_COMPILED_TESTS_SH=`cat ${SCRIPT_DIR}/run_compiled_tests.sh`
echo ${CM_DATA_KUBECONFIG}
echo ${CM_DATA_RUN_COMPILED_TESTS_SH}

kubectl apply -f - <<-EOF
kind: ConfigMap
apiVersion: v1
metadata:
  name: test-config
  namespace: ${JOB_NAMESPACE}
data:
  kubeconfig: |
    ${CM_DATA_KUBECONFIG}
  run_compiled_tests.sh: |
    ${CM_DATA_RUN_COMPILED_TESTS_SH}
EOF

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
        - name: test-runner
          image: ${JOB_IMAGE}
          volumeMounts:
            - name: test-config
              mountPath: /test-config
            - name: ocisecret
              mountPath: /etc/ocisecret
          command: [ "sh", "-c", "sleep 5s;/test-config/run_compiled_tests.sh -t ${TEST_BINARY} -c /test-config/kubeconfig -o ${TEST_LOG_BUCKET} -r ${TEST_REGEX} -l ${TEST_LOG_ARCHIVE}" ]
      restartPolicy: Never
      volumes:
      - name: test-config
        projected:
           sources:
            - configMap:
                name: test-config
                items:
                  - key: kubeconfig
                    path: kubeconfig
                    mode: 0755
                  - key: run_compiled_tests.sh
                    path: run_compiled_tests.sh
                    mode: 0755
      - name: ocisecret
        secret:
          secretName: oci
          optional: true
EOF

kubectl logs -n ${JOB_NAMESPACE} \
    -f $(kubectl get pod \
    -n ${JOB_NAMESPACE} \
    -l job-name=${JOB_NAME} \
    -o jsonpath="{.items[0].metadata.name}")
