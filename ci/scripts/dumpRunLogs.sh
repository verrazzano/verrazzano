# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

set -u

POST_DUMP_FAILED_FILE="${POST_DUMP_FAILED_FILE:-${WORKSPACE}/post_dump_failed_file.tmp}"
VZ_LOGS_ROOT="${VZ_LOGS_ROOT:-${WORKSPACE}/logs}"
VERRAZZANO_INSTALL_LOGS_DIR="${VERRAZZANO_INSTALL_LOGS_DIR:-${VZ_LOGS_ROOT}}"
VZ_COMMAND="${VZ_COMMAND:-${GOPATH}/bin/vz}"
TESTS_EXECUTED_FILE="${TESTS_EXECUTED_FILE:-${WORKSPACE}/tests_executed_file.tmp}"

dumpK8sCluster() {
  ANALYSIS_REPORT="analysis.report"
  dumpDirectory=$1
  ${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d ${dumpDirectory} -r ${dumpDirectory}/cluster-snapshot/${ANALYSIS_REPORT}

  # TODO: Handle any error in creating the bug-report or running analyze on that
  # Create a bug-report and run analysis tool on the bug-report
  # Requires environment variable KUBECONFIG or $HOME/.kube/config
  BUG_REPORT_FILE="${dumpDirectory}/bug-report.tar.gz"
  if [[ -x ${VZ_COMMAND} ]]; then
    $GOPATH/vz bug-report --report-file ${BUG_REPORT_FILE}
  else
    GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go bug-report --report-file ${BUG_REPORT_FILE}
  fi

  # Check if the bug-report exists
  if [ -f "${BUG_REPORT_FILE}" ]; then
    mkdir -p ${dumpDirectory}/bug-report
    tar -xvf ${BUG_REPORT_FILE} -C ${dumpDirectory}/bug-report
    rm ${BUG_REPORT_FILE} || true

    # Run vz analyze on the extracted directory
    if [[ -x ${VZ_COMMAND} ]]; then
      ${VZ_COMMAND} analyze --capture-dir ${dumpDirectory}/bug-report --report-format detailed --report-file ${dumpDirectory}/bug-report/${ANALYSIS_REPORT}
    else
      GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go analyze --capture-dir ${dumpDirectory}/bug-report --report-format detailed --report-file ${dumpDirectory}/bug-report/${ANALYSIS_REPORT}
    fi
  fi
}

dumpVerrazzanoSystemPods() {
  cd ${GO_REPO_PATH}/verrazzano/platform-operator
  local sysLogs=${VERRAZZANO_INSTALL_LOGS_DIR}/system
  mkdir -p ${sysLogs}
  export DIAGNOSTIC_LOG="${sysLogs}/verrazzano-system-pods.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -m "verrazzano system pods" || echo "failed" > ${POST_DUMP_FAILED_FILE}
  export DIAGNOSTIC_LOG="${sysLogs}/verrazzano-system-certs.log"
  ./scripts/install/k8s-dump-objects.sh -o cert -n verrazzano-system -m "verrazzano system certs" || echo "failed" > ${POST_DUMP_FAILED_FILE}
  export DIAGNOSTIC_LOG="${sysLogs}/verrazzano-system-osd.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "vmi-system-osd-*" -m "verrazzano system opensearchdashboards log" -l -c osd || echo "failed" > ${POST_DUMP_FAILED_FILE}
  export DIAGNOSTIC_LOG="${sysLogs}/verrazzano-system-os-master.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "vmi-system-os-master-*" -m "verrazzano system opensearchdashboards log" -l -c es-master || echo "failed" > ${POST_DUMP_FAILED_FILE}
}

dumpCattleSystemPods() {
  cd ${GO_REPO_PATH}/verrazzano/platform-operator
  local rancherLogs=${VERRAZZANO_INSTALL_LOGS_DIR}/rancher
  mkdir -p ${rancherLogs}
  export DIAGNOSTIC_LOG="${rancherLogs}/cattle-system-pods.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n cattle-system -m "cattle system pods" || echo "failed" > ${POST_DUMP_FAILED_FILE}
  export DIAGNOSTIC_LOG="${rancherLogs}/rancher.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n cattle-system -r "rancher-*" -m "Rancher logs" -c rancher -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
}

dumpNginxIngressControllerLogs() {
  cd ${GO_REPO_PATH}/verrazzano/platform-operator
  local nginxLogs=${VERRAZZANO_INSTALL_LOGS_DIR}/nginx
  mkdir -p ${nginxLogs}
  export DIAGNOSTIC_LOG="${nginxLogs}/nginx-ingress-controller.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n ingress-nginx -r "nginx-ingress-controller-*" -m "Nginx Ingress Controller" -c controller -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
}

dumpVerrazzanoPlatformOperatorLogs() {
  ## dump out verrazzano-platform-operator logs
  local vpoLogs=${VZ_LOGS_ROOT}/verrazzano-platform-operator
  mkdir -p ${vpoLogs}
  kubectl -n verrazzano-install logs --selector=app=verrazzano-platform-operator > ${vpoLogs}/verrazzano-platform-operator-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
  kubectl -n verrazzano-install describe pod --selector=app=verrazzano-platform-operator > ${vpoLogs}/verrazzano-platform-operator-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
  echo "verrazzano-platform-operator logs dumped to verrazzano-platform-operator-pod.log"
  echo "verrazzano-platform-operator pod description dumped to verrazzano-platform-operator-pod.out"
  echo "------------------------------------------"
}

dumpVerrazzanoApplicationOperatorLogs() {
  ## dump out verrazzano-application-operator logs
  local vaoLogs=${VZ_LOGS_ROOT}/verrazzano-application-operator
  mkdir -p ${vaoLogs}
  kubectl -n verrazzano-system logs --selector=app=verrazzano-application-operator > ${vaoLogs}/verrazzano-application-operator-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
  kubectl -n verrazzano-system describe pod --selector=app=verrazzano-application-operator > ${vaoLogs}/verrazzano-application-operator-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
  echo "verrazzano-application-operator logs dumped to verrazzano-application-operator-pod.log"
  echo "verrazzano-application-operator pod description dumped to verrazzano-application-operator-pod.out"
  echo "------------------------------------------"
}

dumpOamKubernetesRuntimeLogs() {
  ## dump out oam-kubernetes-runtime logs
  local oamLogs=${VZ_LOGS_ROOT}/oam-kubernetes-runtime
  mkdir -p ${oamLogs}
  kubectl -n verrazzano-system logs --selector=app.kubernetes.io/instance=oam-kubernetes-runtime > ${oamLogs}/oam-kubernetes-runtime-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
  kubectl -n verrazzano-system describe pod --selector=app.kubernetes.io/instance=oam-kubernetes-runtime > ${oamLogs}/oam-kubernetes-runtime-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
  echo "verrazzano-application-operator logs dumped to oam-kubernetes-runtime-pod.log"
  echo "verrazzano-application-operator pod description dumped to oam-kubernetes-runtime-pod.out"
  echo "------------------------------------------"
}

dumpVerrazzanoApiLogs() {
  cd ${GO_REPO_PATH}/verrazzano/platform-operator
  local sysLogs=${VERRAZZANO_INSTALL_LOGS_DIR}/system
  mkdir -p ${sysLogs}
  export DIAGNOSTIC_LOG="${sysLogs}/verrazzano-authproxy.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "verrazzano-authproxy-*" -m "verrazzano api" -c verrazzano-authproxy -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
}

if [ -e ${TESTS_EXECUTED_FILE} ]; then
  dumpVerrazzanoSystemPods
  dumpCattleSystemPods
  dumpNginxIngressControllerLogs
  dumpVerrazzanoPlatformOperatorLogs
  dumpVerrazzanoApplicationOperatorLogs
  dumpOamKubernetesRuntimeLogs
  dumpVerrazzanoApiLogs
fi
