# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

captureFullCluster() {
  # TODO: Capture full cluster only when an environment variable CAPTURE_FULL_CLUSTER is set
  mkdir -p ${FULL_CLUSTER_DIR}
  ${CLUSTER_SNAPSHOT_SCRIPT} -d ${FULL_CLUSTER_DIR} -r ${FULL_CLUSTER_DIR}/${ANALYSIS_REPORT}
}

captureBugReport() {
  # TODO: Handle any error in creating the bug-report or running analyze on that
  # Create a bug-report and run analysis tool on the bug-report
  # Requires environment variable KUBECONFIG or $HOME/.kube/config
  mkdir -p ${BUG_REPORT_DIR}
  $VZ_COMMAND bug-report --report-file ${BUG_REPORT_FILE}

  # Check if the bug-report exists
  if [ -f "${BUG_REPORT_FILE}" ]; then
    tar -xf ${BUG_REPORT_FILE} -C ${BUG_REPORT_DIR}
    rm ${BUG_REPORT_FILE} || true

    # Run vz analyze on the extracted directory
    $VZ_COMMAND analyze --capture-dir ${BUG_REPORT_DIR} --report-file ${BUG_REPORT_DIR}/${ANALYSIS_REPORT} --report-format detailed
  fi
}

if [ -z "$VZ_COMMAND" ]; then
  echo "This script requires an environment variable VZ_COMMAND to indicate the Verrazzano command-line executable"
  exit 1
fi

if [ -z $1 ]; then
    echo "Directory to place the cluster resources is required"
    exit 1
fi

if [ -z $2 ]; then
    echo "The script to capture the cluster resources is required"
    exit 1
fi

ANALYSIS_REPORT="analysis.report"
BUG_REPORT="bug-report.tar.gz"
CLUSTER_SNAPSHOT_ROOT=$1
CLUSTER_SNAPSHOT_SCRIPT=$2

if [ ! -f "${CLUSTER_SNAPSHOT_SCRIPT}" ]; then
  echo "The script to capture the cluster resources ${CLUSTER_SNAPSHOT_SCRIPT} doesn't exist"
  exit 1
fi

FULL_CLUSTER_DIR=${CLUSTER_SNAPSHOT_ROOT}/full-cluster
BUG_REPORT_DIR=${CLUSTER_SNAPSHOT_ROOT}/bug-report
BUG_REPORT_FILE="${BUG_REPORT_DIR}/${BUG_REPORT}"

captureFullCluster
captureBugReport
