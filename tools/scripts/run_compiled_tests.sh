#!/usr/bin/env bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# This script can be used to execute a compiled test binary against a verrazzano cluster, extract and send the metrics to prometheus, 
# and (optionally) push the test logs and report to objectstorage.

SCRIPT_DIR=$(cd $(dirname "$0"); pwd -P)

function usage {
    echo
    echo "Usage:"
    echo " $0 [-f TEST_BINARIES]"
    echo "Arguments:"
    echo "  -f, --test-binaries           The path to the compiled test file or the folder containing compiled test binaries."
    echo "  -p, --prometheus-pushgateway  (Optional) URL to prometheus pushgateway for pushing metrics. It MUST not have a trailing slash ('/')." 
    echo "  -o, --prometheus-job          (Optional) Job name for prometheus metrics."
    echo "  -m, --prometheus-instace      (Optional) Instace name for prometheus metrics."
    echo "  -u, --sauron-cred             (Optional) Sauron credentials in the format: <username>:<password>"
    echo "  -b, --oci-bucket              (Optional) Test log bucket. Required for pushing logs to object storage. Please make sure OCI CLI env variables are set."
    echo "  -k, --kubeconfig-text         (Optional) Kubeconfig text."
    echo "  -c, --kubeconfig-location     (Optional) The url or path to the kubeconfig file."
    echo "  -r, --regex-filter            (Optional) The regex filter for tests used by Ginkgo."
    echo "  -l, --test-log-archive        (Optional) The name of test log archive file."
    echo "  -d, --sleep-duration          (Optional) Time to sleep after every iteration, default is 1 minute."
    echo "  -v, --verbose                 (Optional) Turn on verbose output."
    echo "  -h, --help                    Print usage information."
    echo
    exit 1
}

function is_url {
    regex='^(https?|ftp|file)://[-A-Za-z0-9\+&@#/%?=~_|!:,.;]*[-A-Za-z0-9\+&@#/%=~_|]\.[-A-Za-z0-9\+&@#/%?=~_|!:,.;]*[-A-Za-z0-9\+&@#/%=~_|]$'
    [[ $1 =~ $regex ]]
}

function file_exists_at_url {
    wget --spider $1 2>/dev/null
}

function file_exists {
    [[ -f "$1" && -s "$1" ]]
}

function print {
    if [ ${VERBOSE} ]
    then
        echo "$1" | tee -a "${TEST_LOG_FILE}"
    else
        echo "$1" | cat >> "${TEST_LOG_FILE}"
    fi
}

function execute_tests_recursively {
    for FILE_NAME in `ls $1`
    do
        local FILE_PATH="$1/${FILE_NAME}"
        local FILE_NAME_NO_EXTENSION=${FILE_NAME%.*}
        local EXTENSION=${FILE_PATH##*.}
        if [ -d "${FILE_PATH}" ]
        then 
            execute_tests_recursively ${FILE_PATH}
        elif file_exists "${FILE_PATH}" && [[ -x ${FILE_PATH} && ${EXTENSION} = "test" ]]
        then
            local LOG_FILE="/tmp/${TEST_LOG_ARCHIVE}/${FILE_NAME_NO_EXTENSION}.log"
            ginkgo -v --no-color --keep-going --trace --focus="${TEST_REGEX}" ${FILE_PATH} > ${LOG_FILE} &
            local PID="$!"
            PIDS+=(${PID})
            LOG_FILES+=(${LOG_FILE})
            print "INFO: Running test file (PID: ${PID}): ${FILE_PATH}"
            print "INFO: Created log file: ${LOG_FILE}"
        fi
    done
}

function update_prometheus_metricsfile {
    local METRICS=($(cat ${TEST_TEMP_FILE} | grep -E 'Ran [0-9]+ of [0-9]+ Specs' | awk -F ' ' '{sum_specs_run+=$2; sum_specs_total+=$4;} END {print sum_specs_run, sum_specs_total;}'))
    echo "#HELP lre_tests_specs_total Total number of specs in the test suites" > ${TEST_METRICS_FILE}
    echo "#TYPE lre_tests_specs_total gauge" >> ${TEST_METRICS_FILE}
    echo "lre_tests_specs_total ${METRICS[1]}" >> ${TEST_METRICS_FILE}
    
    echo "#HELP lre_tests_specs_run Total number of specs run in the test suites" >> ${TEST_METRICS_FILE}
    echo "#TYPE lre_tests_specs_run gauge" >> ${TEST_METRICS_FILE}
    echo "lre_tests_specs_run ${METRICS[0]}" >> ${TEST_METRICS_FILE}

    METRICS=($(cat ${TEST_TEMP_FILE} | grep -E '(SUCCESS|FAIL)! -- [0-9]+ Passed \| [0-9]+ Failed \| [0-9]+ Pending \| [0-9]+ Skipped' | awk '{split($0, a, "! -- "); gsub(/\|/, "", a[2]); print a[2]}' | awk -F ' ' '{sum_passed+=$1; sum_failed+=$3; sum_pending+=$5; sum_skipped+=$7}  END {print sum_passed, sum_failed, sum_pending, sum_skipped;}')) 
    echo "#HELP lre_tests_specs_passed Total number of specs passed" >> ${TEST_METRICS_FILE}
    echo "#TYPE lre_tests_specs_passed gauge" >> ${TEST_METRICS_FILE}
    echo "lre_tests_specs_passed ${METRICS[0]}" >> ${TEST_METRICS_FILE}
    
    echo "#HELP lre_tests_specs_failed Total number of specs failed" >> ${TEST_METRICS_FILE}
    echo "#TYPE lre_tests_specs_failed gauge" >> ${TEST_METRICS_FILE}
    echo "lre_tests_specs_failed ${METRICS[1]}" >> ${TEST_METRICS_FILE}
    
    echo "#HELP lre_tests_specs_pending Total number of specs pending" >> ${TEST_METRICS_FILE}
    echo "#TYPE lre_tests_specs_pending gauge" >> ${TEST_METRICS_FILE}
    echo "lre_tests_specs_pending ${METRICS[2]}" >> ${TEST_METRICS_FILE}
    
    echo "#HELP lre_tests_specs_skipped Total number of specs skipped" >> ${TEST_METRICS_FILE}
    echo "#TYPE lre_tests_specs_skipped gauge" >> ${TEST_METRICS_FILE}
    echo "lre_tests_specs_skipped ${METRICS[3]}" >> ${TEST_METRICS_FILE}

    METRICS=$(cat ${TEST_TEMP_FILE} | grep -E 'Ginkgo ran [0-9]+ suite' | awk -F ' ' '{sum_suite+=$3;} END {print sum_suite;}')
    echo "#HELP lre_tests_suites_run Total number of test suites run" >> ${TEST_METRICS_FILE}
    echo "#TYPE lre_tests_suites_run gauge" >> ${TEST_METRICS_FILE}
    echo "lre_tests_suites_run ${METRICS}" >> ${TEST_METRICS_FILE}
}

TEST_BINARIES=""
PROMETHEUS_GW_URL=""
PROMETHEUS_JOB=""
PROMETHEUS_INSTANCE=""
PROMETHEUS_PUSH_METRICS=""
SAURON_CRED=""
KUBECONFIG_TEXT=""
KUBECONFIG_LOCATION=""
OCI_BUCKET=""
OCI_CONFIG_LOCATION="/var/run/secrets/ocisecret/oci.yaml"
OCI_PUSH_LOGS=false
TEST_REGEX=""
TEST_REGEX_DEFAULT=".*"
TEST_LOG_ARCHIVE=""
TS=`date "+%Y%m%d-%H%M%S%ss"`
TEST_LOG_ARCHIVE_DEFAULT="lre-tests-logs-${TS}"
VERBOSE=""
SLEEP_DURATION=""
SLEEP_DURATION_DEFAULT="1m"

# Parse the command line arguments
while [ $# -gt 0 ]
do
    key="$1"
    case ${key} in 
        -f|--test-binaries)            TEST_BINARIES="${2%\/}"; shift; shift;;
        -b|--oci-bucket)               OCI_BUCKET="$2"; shift; shift;;
        -p|--prometheus-pushgateway)   PROMETHEUS_GW_URL="$2"; shift; shift;;
        -o|--prometheus-job)           PROMETHEUS_JOB="$2"; shift; shift;;
        -m|--prometheus-instance)      PROMETHEUS_INSTANCE="$2"; shift; shift;;
        -u|--sauron-cred)              SAURON_CRED="$2"; shift; shift;;
        -k|--kubeconfig-text)          KUBECONFIG_TEXT="$2"; shift; shift;;
        -c|--kubeconfig-location)      KUBECONFIG_LOCATION="$2"; shift; shift;;
        -r|--regex-filter)             TEST_REGEX="$2"; shift; shift;;
        -l|--test-log-archive)         TEST_LOG_ARCHIVE="$2"; shift; shift;;
        -d|--sleep-duration)           SLEEP_DURATION="$2"; shift; shift;;
        -v|--verbose)                  VERBOSE=true; shift;;
        -h|--help)                     usage;;
        *)                             echo "ERROR: Invalid argument: ${key}"; usage;;
    esac
done

if [ -z "${TEST_BINARIES}" ]
then
    echo "ERROR: Missing required argument: -f, --test-binaries"
    usage
elif [[ ! -d "${TEST_BINARIES}" ]] && ! file_exists "${TEST_BINARIES}"
then 
    echo "ERROR: No such file or folder exists: ${TEST_BINARIES}"
    exit 1
elif [[ ! -z "${PROMETHEUS_GW_URL}" || ! -z "${PROMETHEUS_JOB}" || ! -z "${PROMETHEUS_INSTANCE}" || ! -z "${SAURON_CRED}" ]]
then
    if [ -z "${PROMETHEUS_GW_URL}" ]
    then
        echo "ERROR: Missing required argument: -p, --prometheus-pushgateway"
        usage
    elif [ -z "${PROMETHEUS_JOB}" ]
    then
        echo "ERROR: Missing required argument: -o, --prometheus-job"
        usage
    elif [ -z "${PROMETHEUS_INSTANCE}" ]
    then
        echo "ERROR: Missing required argument: -m, --prometheus-instance"
        usage
    elif [ -z "${SAURON_CRED}" ]
    then
        echo "ERROR: Missing required argument: -u, --sauron-cred"
        usage
    else
        PROMETHEUS_PUSH_METRICS=true
    fi
fi

if [ -z "${TEST_REGEX}" ]
then
    TEST_REGEX="${TEST_REGEX_DEFAULT}"
fi

if [ -z "${SLEEP_DURATION}" ]
then
    SLEEP_DURATION="${SLEEP_DURATION_DEFAULT}"
fi


# Fetch kubeconfig from the given URL
if is_url "${KUBECONFIG_LOCATION}"
then
    if file_exists_at_url "${KUBECONFIG_LOCATION}"
    then
        local FILE_NAME=`basename "${KUBECONFIG_LOCATION}"`
        rm -rf /tmp/${FILE_NAME}
        wget "${KUBECONFIG_LOCATION}" -O "/tmp/${FILE_NAME}"
        KUBECONFIG_LOCATION="/tmp/${FILE_NAME}"
    else
        echo "ERROR: No file exists at the URL specified: ${KUBECONFIG_LOCATION}"
        exit 1
    fi
fi

# Get kubeconfig from local path
if file_exists "${KUBECONFIG_LOCATION}"
then
    echo "INFO: Found Kubeconfig: ${KUBECONFIG_LOCATION}"
else
    if [ -z "${KUBECONFIG_TEXT}" ]
    then
        if [ ! -z "${KUBECONFIG}" ]
        then
            echo "INFO: Using KUBECONFIG value from env variable: ${KUBECONFIG}"
            KUBECONFIG_LOCATION="${KUBECONFIG}"
        else
            echo "ERROR: Either provide --kubeconfig-text, or a valid --kubeconfig-location, or set the KUBECONFIG env variable to a valid kubeconfig"
            exit 1
        fi
    else
        KUBECONFIG_LOCATION="/tmp/kubeconfig"
        rm -rf "${KUBECONFIG_LOCATION}"
        touch "${KUBECONFIG_LOCATION}"
        echo "${KUBECONFIG_TEXT}" > "${KUBECONFIG_LOCATION}"
    fi
fi

chmod a+x "${KUBECONFIG_LOCATION}"
export KUBECONFIG="${KUBECONFIG_LOCATION}"

# Set OCI config env variables 
if [ ! -z "${OCI_BUCKET}" ]
then
    if [ ! -f "${OCI_CONFIG_LOCATION}" ]
    then
        echo "ERROR: OCI config file does not exist at: ${OCI_CONFIG_LOCATION}"
        exit 1
    else
        echo "INFO: Setting OCI env variables"
        export OCI_CLI_PROFILE="DEFAULT"
        export OCI_CLI_USER=`yq e .auth.user ${OCI_CONFIG_LOCATION}`
        export OCI_CLI_FINGERPRINT=`yq e .auth.fingerprint ${OCI_CONFIG_LOCATION}`
        yq e .auth.key ${OCI_CONFIG_LOCATION} > /tmp/ocikey.pem
        export OCI_CLI_KEY_FILE=/tmp/ocikey.pem
        export OCI_CLI_TENANCY=`yq e .auth.tenancy ${OCI_CONFIG_LOCATION}`
        export OCI_CLI_REGION=`yq e .auth.region ${OCI_CONFIG_LOCATION}`
        export OCI_CLI_SUPPRESS_FILE_PERMISSIONS_WARNING="True"
        OCI_PUSH_LOGS=true
    fi
fi

# Create directory for logs 
if [ -z "${TEST_LOG_ARCHIVE}" ]
then
    TEST_LOG_ARCHIVE=${TEST_LOG_ARCHIVE_DEFAULT}
fi
rm -rf "/tmp/${TEST_LOG_ARCHIVE}"
mkdir -p "/tmp/${TEST_LOG_ARCHIVE}"

TEST_LOG_FILE="/tmp/${TEST_LOG_ARCHIVE}/logs.txt"
TEST_TEMP_FILE="/tmp/${TEST_LOG_ARCHIVE}/tmp.txt"
TEST_METRICS_FILE="/tmp/${TEST_LOG_ARCHIVE}/metrics.txt"
echo -n "" > ${TEST_LOG_FILE}
echo -n "" > ${TEST_TEMP_FILE}
echo -n "" > ${TEST_METRICS_FILE}

# Run the tests
ITERATION=0
while true
do
    print "----------------------------------"
    print "INFO: Iteration: ${ITERATION}"

    # Array to keep track of process IDs of the spawned background processes
    PIDS=()
    LOG_FILES=()

    # Execute tests
    execute_tests_recursively ${TEST_BINARIES}

    if [[ ${#PIDS[@]} -eq 0 ]]
    then
        print "INFO: No test files found."
    else
        # Wait for all the spawned background processes to finish
        print "INFO: Waiting for the tests to finish ..."
        for PID in ${PIDS[@]}
        do 
            wait ${PID}
        done

        # Extract metrics from the logfiles
        for LOG_FILE in ${LOG_FILES[@]}
        do
            cat ${LOG_FILE} \
            | grep -E 'Running Suite:|Ran [0-9]+ of [0-9]+ Specs|(SUCCESS|FAIL)! -- [0-9]+ Passed \| [0-9]+ Failed \| [0-9]+ Pending \| [0-9]+ Skipped|Ginkgo ran [0-9]+ suite' \
            >> ${TEST_TEMP_FILE}
        done
        cat ${TEST_TEMP_FILE} >> ${TEST_LOG_FILE}

        # Send metrics to prometheus pushgateway
        if [ "${PROMETHEUS_PUSH_METRICS}" = true ]
        then
            # Update prometheus metrics file
            print "INFO: Updating prometheus metrics file: ${TEST_METRICS_FILE}"
            update_prometheus_metricsfile
            print "INFO: Sending metrics to prometheus pushgateway: ${PROMETHEUS_GW_URL}/metrics/job/${PROMETHEUS_JOB}"
            cat ${TEST_METRICS_FILE} \
            | curl -i --data-binary @- ${PROMETHEUS_GW_URL}/metrics/job/${PROMETHEUS_JOB}/instance/${PROMETHEUS_INSTANCE} -u ${SAURON_CRED}
        fi

        # Push logs to object store
        if [ "${OCI_PUSH_LOGS}" = true ]
        then
            print "INFO: Compressing directory /tmp/${TEST_LOG_ARCHIVE}"
            tar -czvf "/tmp/${TEST_LOG_ARCHIVE}.tgz" "/tmp/${TEST_LOG_ARCHIVE}"
            print "INFO: Pushing ${TEST_LOG_ARCHIVE}.tgz to object storage"
            yes | oci os object put --bucket-name "${OCI_BUCKET}" --file "/tmp/${TEST_LOG_ARCHIVE}.tgz"
        fi
    fi

    echo -n "" > ${TEST_TEMP_FILE}
    ((ITERATION++))

    # Sleep after every iteration
    print "INFO: Sleeping for ${SLEEP_DURATION}"
    print "----------------------------------"
    sleep ${SLEEP_DURATION}
done
