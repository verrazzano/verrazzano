# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

dumpK8sCluster() {
  ANALYSIS_REPORT="analysis.report"
  dumpDirectory=$1
  ${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d ${dumpDirectory} -r ${dumpDirectory}/cluster-snapshot/${ANALYSIS_REPORT}

  # TODO: Handle any error in creating the bug-report or running analyze on that
  # Create a bug-report and run analysis tool on the bug-report
  # Requires environment variable KUBECONFIG or $HOME/.kube/config
  BUG_REPORT_FILE="${dumpDirectory}/bug-report.tar.gz"
  if [[ -x $GOPATH/bin/vz ]]; then
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
    if [[ -x $GOPATH/bin/vz ]]; then
      $GOPATH/vz analyze --capture-dir ${dumpDirectory}/bug-report --report-format detailed --report-file ${dumpDirectory}/bug-report/${ANALYSIS_REPORT}
    else
      GO111MODULE=on GOPRIVATE=github.com/verrazzano go run main.go analyze --capture-dir ${dumpDirectory}/bug-report --report-format detailed --report-file ${dumpDirectory}/bug-report/${ANALYSIS_REPORT}
    fi
  fi
}

dumpVerrazzanoSystemPods() {
  cd ${GO_REPO_PATH}/verrazzano/platform-operator
  export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-system-pods.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -m "verrazzano system pods" || echo "failed" > ${POST_DUMP_FAILED_FILE}
  export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-system-certs.log"
  ./scripts/install/k8s-dump-objects.sh -o cert -n verrazzano-system -m "verrazzano system certs" || echo "failed" > ${POST_DUMP_FAILED_FILE}
  export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-system-kibana.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "vmi-system-kibana-*" -m "verrazzano system kibana log" -l -c kibana || echo "failed" > ${POST_DUMP_FAILED_FILE}
  export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-system-es-master.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "vmi-system-es-master-*" -m "verrazzano system kibana log" -l -c es-master || echo "failed" > ${POST_DUMP_FAILED_FILE}
}

dumpCattleSystemPods() {
  cd ${GO_REPO_PATH}/verrazzano/platform-operator
  export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/cattle-system-pods.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n cattle-system -m "cattle system pods" || echo "failed" > ${POST_DUMP_FAILED_FILE}
  export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/rancher.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n cattle-system -r "rancher-*" -m "Rancher logs" -c rancher -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
}

dumpNginxIngressControllerLogs() {
  cd ${GO_REPO_PATH}/verrazzano/platform-operator
  export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/nginx-ingress-controller.log"
  ./scripts/install/k8s-dump-objects.sh -o pods -n ingress-nginx -r "nginx-ingress-controller-*" -m "Nginx Ingress Controller" -c controller -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
}

dumpVerrazzanoPlatformOperatorLogs() {
  ## dump out verrazzano-platform-operator logs
  mkdir -p ${WORKSPACE}/verrazzano-platform-operator/logs
  kubectl -n verrazzano-install logs --selector=app=verrazzano-platform-operator > ${WORKSPACE}/verrazzano-platform-operator/logs/verrazzano-platform-operator-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
  kubectl -n verrazzano-install describe pod --selector=app=verrazzano-platform-operator > ${WORKSPACE}/verrazzano-platform-operator/logs/verrazzano-platform-operator-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
  echo "verrazzano-platform-operator logs dumped to verrazzano-platform-operator-pod.log"
  echo "verrazzano-platform-operator pod description dumped to verrazzano-platform-operator-pod.out"
  echo "------------------------------------------"
}

dumpVerrazzanoApplicationOperatorLogs() {
  ## dump out verrazzano-application-operator logs
  mkdir -p ${WORKSPACE}/verrazzano-application-operator/logs
  kubectl -n verrazzano-system logs --selector=app=verrazzano-application-operator > ${WORKSPACE}/verrazzano-application-operator/logs/verrazzano-application-operator-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
  kubectl -n verrazzano-system describe pod --selector=app=verrazzano-application-operator > ${WORKSPACE}/verrazzano-application-operator/logs/verrazzano-application-operator-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
  echo "verrazzano-application-operator logs dumped to verrazzano-application-operator-pod.log"
  echo "verrazzano-application-operator pod description dumped to verrazzano-application-operator-pod.out"
  echo "------------------------------------------"
}

dumpOamKubernetesRuntimeLogs() {
  ## dump out oam-kubernetes-runtime logs
  mkdir -p ${WORKSPACE}/oam-kubernetes-runtime/logs
  kubectl -n verrazzano-system logs --selector=app.kubernetes.io/instance=oam-kubernetes-runtime > ${WORKSPACE}/oam-kubernetes-runtime/logs/oam-kubernetes-runtime-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
  kubectl -n verrazzano-system describe pod --selector=app.kubernetes.io/instance=oam-kubernetes-runtime > ${WORKSPACE}/verrazzano-application-operator/logs/oam-kubernetes-runtime-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
  echo "verrazzano-application-operator logs dumped to oam-kubernetes-runtime-pod.log"
  echo "verrazzano-application-operator pod description dumped to oam-kubernetes-runtime-pod.out"
  echo "------------------------------------------"
}

dumpVerrazzanoApiLogs() {
  cd ${GO_REPO_PATH}/verrazzano/platform-operator
  export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-authproxy.log"
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
