// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

def DOCKER_IMAGE_TAG
def SKIP_ACCEPTANCE_TESTS = false
def SKIP_TRIGGERED_TESTS = false
def SUSPECT_LIST = ""

def agentLabel = env.JOB_NAME.contains('master') ? "phxlarge" : "VM.Standard2.8"

pipeline {
    options {
        skipDefaultCheckout true
    }

    agent {
       docker {
            image "${RUNNER_DOCKER_IMAGE}"
            args "${RUNNER_DOCKER_ARGS}"
            registryUrl "${RUNNER_DOCKER_REGISTRY_URL}"
            registryCredentialsId 'ocir-pull-and-push-account'
            label "${agentLabel}"
        }
    }

    parameters {
        booleanParam (description: 'Whether to kick off acceptance test run at the end of this build', name: 'RUN_ACCEPTANCE_TESTS', defaultValue: true)
        booleanParam (description: 'Whether to include the slow tests in the acceptance tests', name: 'RUN_SLOW_TESTS', defaultValue: false)
        booleanParam (description: 'Whether to create the cluster with Calico for AT testing (defaults to true)', name: 'CREATE_CLUSTER_USE_CALICO', defaultValue: true)
        booleanParam (description: 'Whether to dump k8s cluster on success (off by default can be useful to capture for comparing to failed cluster)', name: 'DUMP_K8S_CLUSTER_ON_SUCCESS', defaultValue: false)
        booleanParam (description: 'Whether to trigger full testing after a successful run. Off by default. This is always done for successful master and release* builds, this setting only is used to enable the trigger for other branches', name: 'TRIGGER_FULL_TESTS', defaultValue: false)
        booleanParam (description: 'Whether to generate a tarball', name: 'GENERATE_TARBALL', defaultValue: false)
        booleanParam (description: 'Whether to generate a tarball', name: 'SIMULATE_FAILURE', defaultValue: false)
        choice (name: 'WILDCARD_DNS_DOMAIN',
                description: 'Wildcard DNS Domain',
                // 1st choice is the default value
                choices: [ "nip.io", "sslip.io"])
    }

    environment {
        DOCKER_ANALYSIS_CI_IMAGE_NAME = 'verrazzano-analysis-jenkins'
        DOCKER_ANALYSIS_PUBLISH_IMAGE_NAME = 'verrazzano-analysis'
        DOCKER_ANALYSIS_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_ANALYSIS_PUBLISH_IMAGE_NAME : env.DOCKER_ANALYSIS_CI_IMAGE_NAME}"
        DOCKER_PLATFORM_CI_IMAGE_NAME = 'verrazzano-platform-operator-jenkins'
        DOCKER_PLATFORM_PUBLISH_IMAGE_NAME = 'verrazzano-platform-operator'
        DOCKER_PLATFORM_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_PLATFORM_PUBLISH_IMAGE_NAME : env.DOCKER_PLATFORM_CI_IMAGE_NAME}"
        DOCKER_OAM_CI_IMAGE_NAME = 'verrazzano-application-operator-jenkins'
        DOCKER_OAM_PUBLISH_IMAGE_NAME = 'verrazzano-application-operator'
        DOCKER_OAM_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_OAM_PUBLISH_IMAGE_NAME : env.DOCKER_OAM_CI_IMAGE_NAME}"
        CREATE_LATEST_TAG = "${env.BRANCH_NAME == 'master' ? '1' : '0'}"
        GOPATH = '/home/opc/go'
        GO_REPO_PATH = "${GOPATH}/src/github.com/verrazzano"
        DOCKER_CREDS = credentials('github-packages-credentials-rw')
        DOCKER_EMAIL = credentials('github-packages-email')
        DOCKER_REPO = 'ghcr.io'
        DOCKER_NAMESPACE = 'verrazzano'
        NETRC_FILE = credentials('netrc')
        GITHUB_API_TOKEN = credentials('github-api-token-release-assets')
        GITHUB_RELEASE_USERID = credentials('github-userid-release')
        GITHUB_RELEASE_EMAIL = credentials('github-email-release')
        SERVICE_KEY = credentials('PAGERDUTY_SERVICE_KEY')

        CLUSTER_NAME = 'verrazzano'
        POST_DUMP_FAILED_FILE = "${WORKSPACE}/post_dump_failed_file.tmp"
        TESTS_EXECUTED_FILE = "${WORKSPACE}/tests_executed_file.tmp"
        KUBECONFIG = "${WORKSPACE}/test_kubeconfig"
        VERRAZZANO_KUBECONFIG = "${KUBECONFIG}"
        OCR_CREDS = credentials('ocr-pull-and-push-account')
        OCR_REPO = 'container-registry.oracle.com'
        IMAGE_PULL_SECRET = 'verrazzano-container-registry'
        INSTALL_CONFIG_FILE_KIND = "./tests/e2e/config/scripts/install-verrazzano-kind.yaml"
        INSTALL_PROFILE = "dev"
        VZ_ENVIRONMENT_NAME = "default"
        TEST_SCRIPTS_DIR = "${GO_REPO_PATH}/verrazzano/tests/e2e/config/scripts"

        WEBLOGIC_PSW = credentials('weblogic-example-domain-password') // Needed by ToDoList example test
        DATABASE_PSW = credentials('todo-mysql-password') // Needed by ToDoList example test

        JENKINS_READ = credentials('jenkins-auditor')

        OCI_CLI_AUTH="instance_principal"
        OCI_OS_NAMESPACE = credentials('oci-os-namespace')
        OCI_OS_ARTIFACT_BUCKET="build-failure-artifacts"
        OCI_OS_BUCKET="verrazzano-builds"

        // Used for dumping cluster from inside tests
        DUMP_KUBECONFIG="${KUBECONFIG}"
        DUMP_COMMAND="${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh"
        TEST_DUMP_ROOT="${WORKSPACE}/test-cluster-dumps"

        VERRAZZANO_INSTALL_LOGS_DIR="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs"
        VERRAZZANO_INSTALL_LOG="verrazzano-install.log"
    }

    stages {
        stage('Clean workspace and checkout') {
            steps {
                sh """
                    echo "${NODE_LABELS}"
                """

                script {
                    def scmInfo = checkout scm
                    env.GIT_COMMIT = scmInfo.GIT_COMMIT
                    env.GIT_BRANCH = scmInfo.GIT_BRANCH
                    echo "SCM checkout of ${env.GIT_BRANCH} at ${env.GIT_COMMIT}"
                }
                sh """
                    cp -f "${NETRC_FILE}" $HOME/.netrc
                    chmod 600 $HOME/.netrc
                """

                script {
                    try {
                        sh """
                            echo "${DOCKER_CREDS_PSW}" | docker login ${env.DOCKER_REPO} -u ${DOCKER_CREDS_USR} --password-stdin
                        """
                    } catch(error) {
                        echo "docker login failed, retrying after sleep"
                        retry(4) {
                            sleep(30)
                            sh """
                            echo "${DOCKER_CREDS_PSW}" | docker login ${env.DOCKER_REPO} -u ${DOCKER_CREDS_USR} --password-stdin
                            """
                        }
                    }
                }
                script {
                    try {
                        sh """
                            echo "${OCR_CREDS_PSW}" | docker login -u ${OCR_CREDS_USR} ${OCR_REPO} --password-stdin
                        """
                    } catch(error) {
                        echo "OCR docker login failed, retrying after sleep"
                        retry(4) {
                            sleep(30)
                            sh """
                                echo "${OCR_CREDS_PSW}" | docker login -u ${OCR_CREDS_USR} ${OCR_REPO} --password-stdin
                            """
                        }
                    }
                }
                sh """
                    rm -rf ${GO_REPO_PATH}/verrazzano
                    mkdir -p ${GO_REPO_PATH}/verrazzano
                    tar cf - . | (cd ${GO_REPO_PATH}/verrazzano/ ; tar xf -)
                """

                script {
                    def props = readProperties file: '.verrazzano-development-version'
                    VERRAZZANO_DEV_VERSION = props['verrazzano-development-version']
                    TIMESTAMP = sh(returnStdout: true, script: "date +%Y%m%d%H%M%S").trim()
                    SHORT_COMMIT_HASH = sh(returnStdout: true, script: "git rev-parse --short=8 HEAD").trim()
                    DOCKER_IMAGE_TAG = "${VERRAZZANO_DEV_VERSION}-${TIMESTAMP}-${SHORT_COMMIT_HASH}"
                    // update the description with some meaningful info
                    currentBuild.description = SHORT_COMMIT_HASH + " : " + env.GIT_COMMIT
                    def currentCommitHash = env.GIT_COMMIT
                    def commitList = getCommitList()
                    withCredentials([file(credentialsId: 'jenkins-to-slack-users', variable: 'JENKINS_TO_SLACK_JSON')]) {
                        def userMappings = readJSON file: JENKINS_TO_SLACK_JSON
                        SUSPECT_LIST = getSuspectList(commitList, userMappings)
                        echo "Suspect list: ${SUSPECT_LIST}"
                    }
                }
            }
        }

        stage('Analysis Tool') {
            when {
                allOf {
                    not { buildingTag() }
                    anyOf {
                        branch 'master';
                        branch 'release-*';
                    }
                }
            }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano/tools/analysis
                    make go-build
                    cd out
                    zip -r ${WORKSPACE}/analysis-tool.zip linux_amd64 darwin_amd64
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/analysis-tool.zip --file ${WORKSPACE}/analysis-tool.zip
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SHORT_COMMIT_HASH}/analysis-tool.zip --file ${WORKSPACE}/analysis-tool.zip
                """
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                always {
                    archiveArtifacts artifacts: '**/analysis-tool.zip', allowEmptyArchive: true
                }
            }
        }

        stage('Generate operator.yaml') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano/platform-operator
                    case "${env.BRANCH_NAME}" in
                        master|release-*)
                            ;;
                        *)
                            echo "Adding image pull secrets to operator.yaml for non master/release branch"
                            export IMAGE_PULL_SECRETS=verrazzano-container-registry
                    esac
                    DOCKER_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} OPERATOR_YAML=$WORKSPACE/generated-operator.yaml make generate-operator-yaml
                   """
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                success {
                    archiveArtifacts artifacts: "generated-operator.yaml", allowEmptyArchive: true
                }
            }
        }

        stage('Build') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano
                    make docker-push VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} VERRAZZANO_ANALYSIS_IMAGE_NAME=${DOCKER_ANALYSIS_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
                    cp ${GO_REPO_PATH}/verrazzano/platform-operator/out/generated-verrazzano-bom.json $WORKSPACE/generated-verrazzano-bom.json
                   """
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                success {
                    archiveArtifacts artifacts: "generated-verrazzano-bom.json", allowEmptyArchive: true
                }
            }
        }

        stage('Save Generated Files') {
            when {
                allOf {
                    not { buildingTag() }
                }
            }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/operator.yaml --file $WORKSPACE/generated-operator.yaml
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SHORT_COMMIT_HASH}/operator.yaml --file $WORKSPACE/generated-operator.yaml
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/generated-verrazzano-bom.json --file $WORKSPACE/generated-verrazzano-bom.json
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SHORT_COMMIT_HASH}/generated-verrazzano-bom.json --file $WORKSPACE/generated-verrazzano-bom.json
                   """
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
            }
        }

        stage('Quality and Compliance Checks') {
            when { not { buildingTag() } }
            steps {
                sh """
                    echo "run all linters"
                    cd ${GO_REPO_PATH}/verrazzano
                    make check

                    echo "copyright scan"
                    time make copyright-check

                    echo "Third party license check"
                """
                thirdpartyCheck()
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
            }
        }

        stage('Unit Tests') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano
                    make -B coverage
                """
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                always {
                    sh """
                        cd ${GO_REPO_PATH}/verrazzano
                        cp coverage.html ${WORKSPACE}
                        cp coverage.xml ${WORKSPACE}
                        build/copy-junit-output.sh ${WORKSPACE}
                    """
                    archiveArtifacts artifacts: '**/coverage.html', allowEmptyArchive: true
                    junit testResults: '**/*test-result.xml', allowEmptyResults: true
                    cobertura(coberturaReportFile: 'coverage.xml',
                      enableNewApi: true,
                      autoUpdateHealth: false,
                      autoUpdateStability: false,
                      failUnstable: true,
                      failUnhealthy: true,
                      failNoReports: true,
                      onlyStable: false,
                      fileCoverageTargets: '100, 0, 0',
                      lineCoverageTargets: '75, 75, 75',
                      packageCoverageTargets: '100, 0, 0',
                    )
                }
            }
        }

        stage('Scan Image') {
            when { not { buildingTag() } }
            steps {
                script {
                    clairScanTemp "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_PLATFORM_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
                    clairScanTemp "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_OAM_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
                    clairScanTemp "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_ANALYSIS_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
                }
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                always {
                    archiveArtifacts artifacts: '**/scanning-report.json', allowEmptyArchive: true
                }
            }
        }

        stage('Integration Tests') {
            when { not { buildingTag() } }
            steps {
                sh """
                    if [ "${params.SIMULATE_FAILURE} == "true" ]; then
                        echo "Simulate failure from a stage"
                        exit 1
                    fi
                    cd ${GO_REPO_PATH}/verrazzano/platform-operator

                    make cleanup-cluster
                    make create-cluster KIND_CONFIG="kind-config-ci.yaml"
                    ../ci/scripts/setup_kind_for_jenkins.sh
                    make integ-test CLUSTER_DUMP_LOCATION=${WORKSPACE}/platform-operator-integ-cluster-dump DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG}
                    ../build/copy-junit-output.sh ${WORKSPACE}
                    cd ${GO_REPO_PATH}/verrazzano/application-operator
                    make cleanup-cluster
                    make integ-test KIND_CONFIG="kind-config-ci.yaml" CLUSTER_DUMP_LOCATION=${WORKSPACE}/application-operator-integ-cluster-dump DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG}
                    ../build/copy-junit-output.sh ${WORKSPACE}
                    make cleanup-cluster
                """
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                always {
                    archiveArtifacts artifacts: '**/coverage.html,**/logs/*,**/*-cluster-dump/**', allowEmptyArchive: true
                    junit testResults: '**/*test-result.xml', allowEmptyResults: true
                }
            }
        }

        stage('Skip acceptance tests if commit message contains skip-at') {
            steps {
                script {
                    // note that SKIP_ACCEPTANCE_TESTS will be false at this point (its default value)
                    // so we are going to run the AT's unless this logic decides to skip them...

                    // if we are planning to run the AT's (which is the default)
                    if (params.RUN_ACCEPTANCE_TESTS == true) {
                        SKIP_ACCEPTANCE_TESTS = false
                        // check if the user has asked to skip AT using the commit message
                        result = sh (script: "git log -1 | grep 'skip-at'", returnStatus: true)
                        if (result == 0) {
                            // found 'skip-at', so don't run them
                            SKIP_ACCEPTANCE_TESTS = true
                            echo "Skip acceptance tests based on opt-out in commit message [skip-at]"
                            echo "SKIP_ACCEPTANCE_TESTS is ${SKIP_ACCEPTANCE_TESTS}"
                        }
                    } else {
                        SKIP_ACCEPTANCE_TESTS = true
                    }
                }
            }
        }

        stage('Acceptance Tests') {
            when {
                allOf {
                    not { buildingTag() }
                    anyOf {
                        branch 'master';
                        branch 'release-*';
                        expression {SKIP_ACCEPTANCE_TESTS == false};
                    }
                }
            }

            stages {

                stage('Prepare AT environment') {
                    environment {
                        VERRAZZANO_OPERATOR_IMAGE="NONE"
                        KIND_KUBERNETES_CLUSTER_VERSION="1.18"
                        OCI_OS_LOCATION="${SHORT_COMMIT_HASH}"
                    }
                    steps {
                        sh """
                            cd ${GO_REPO_PATH}/verrazzano
                            ci/scripts/prepare_jenkins_at_environment.sh ${params.CREATE_CLUSTER_USE_CALICO} ${params.WILDCARD_DNS_DOMAIN}
                        """
                    }
                    post {
                        always {
                            archiveArtifacts artifacts: "acceptance-test-operator.yaml,downloaded-operator.yaml", allowEmptyArchive: true
                            sh """
                                ## dump out install logs
                                mkdir -p ${VERRAZZANO_INSTALL_LOGS_DIR}
                                kubectl logs --selector=job-name=verrazzano-install-my-verrazzano > ${VERRAZZANO_INSTALL_LOGS_DIR}/${VERRAZZANO_INSTALL_LOG} --tail -1
                                kubectl describe pod --selector=job-name=verrazzano-install-my-verrazzano > ${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-install-job-pod.out
                                echo "Verrazzano Installation logs dumped to verrazzano-install.log"
                                echo "Verrazzano Install pod description dumped to verrazzano-install-job-pod.out"
                                echo "------------------------------------------"
                            """
                        }
                    }
                }

                stage('Run Acceptance Tests') {
                    environment {
                        TEST_ENV = "KIND"
                    }
                    parallel {
                        stage('verify-install') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/verify-install"
                            }
                            steps {
                                runGinkgoRandomize('verify-install')
                            }
                        }
                        stage('verify-scripts') {
                            steps {
                                runGinkgo('scripts')
                            }
                        }
                        stage('verify-infra restapi') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/verify-infra-restapi"
                            }
                            steps {
                                runGinkgoRandomize('verify-infra/restapi')
                            }
                        }
                        stage('verify-infra oam') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/verify-infra-oam"
                            }
                            steps {
                                runGinkgoRandomize('verify-infra/oam')
                            }
                        }
                        stage('verify-infra vmi') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/verify-infra-vmi"
                            }
                            steps {
                                runGinkgoRandomize('verify-infra/vmi')
                            }
                        }
                        stage('istio authorization policy') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/istio-authz-policy"
                            }
                            steps {
                                runGinkgo('istio/authz')
                            }
                        }
                        stage('security role based access') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/sec-role-based-access"
                            }
                            steps {
                                runGinkgo('security/rbac')
                            }
                        }
                        stage('security network policies') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/sec-network-policies"
                            }
                            steps {
                                script {
                                    if (params.CREATE_CLUSTER_USE_CALICO == true) {
                                        runGinkgo('security/network-policies')
                                    }
                                }
                            }
                        }
                        stage('k8s deployment workload metrics') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/k8sdeploy-workload-metrics"
                            }
                            steps {
                                runGinkgo('deploymetrics')
                            }
                        }
                        stage('examples logging helidon') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/examples-logging-helidon"
                            }
                            steps {
                                runGinkgo('logging/helidon')
                            }
                        }
                        stage('examples todo') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/examples-todo"
                            }
                            steps {
                                runGinkgo('examples/todo')
                            }
                        }
                        stage('examples socks') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/examples-socks"
                            }
                            steps {
                                runGinkgo('examples/socks')
                            }
                        }
                        stage('examples spring') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/examples-spring"
                            }
                            steps {
                                runGinkgo('examples/springboot')
                            }
                        }
                        stage('examples helidon') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/examples-helidon"
                            }
                            steps {
                                runGinkgo('examples/helidon')
                            }
                        }
                        stage('examples helidon-config') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/examples-helidon-config"
                            }
                            steps {
                                runGinkgo('examples/helidonconfig')
                            }
                        }
                        stage('examples bobs') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/examples-bobs"
                            }
                            when {
                                expression {params.RUN_SLOW_TESTS == true}
                            }
                            steps {
                                runGinkgo('examples/bobsbooks')
                            }
                        }
                        stage('console ingress') {
                            environment {
                                DUMP_DIRECTORY="${TEST_DUMP_ROOT}/console-ingress"
                            }
                            steps {
                                runGinkgo('ingress/console')
                            }
                        }
                    }
                    post {
                        always {
                            archiveArtifacts artifacts: '**/coverage.html,**/logs/*,**/test-cluster-dumps/**', allowEmptyArchive: true
                            junit testResults: '**/*test-result.xml', allowEmptyResults: true
                        }
                    }
                }
            }

            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                        if ( fileExists(env.TESTS_EXECUTED_FILE) ) {
                            dumpK8sCluster('new-acceptance-tests-cluster-dump')
                        }
                    }
                }
                success {
                    script {
                        if (params.DUMP_K8S_CLUSTER_ON_SUCCESS == true && fileExists(env.TESTS_EXECUTED_FILE) ) {
                            dumpK8sCluster('new-acceptance-tests-cluster-dump')
                        }
                    }
                }
            }
        }

        stage('Triggered Tests') {
            when {
                allOf {
                    not { buildingTag() }
                    expression {SKIP_TRIGGERED_TESTS == false}
                    anyOf {
                        branch 'master';
                        branch 'release-*';
                        expression {params.TRIGGER_FULL_TESTS == true};
                    }
                }
            }
            steps {
                script {
                    build job: "verrazzano-push-triggered-acceptance-tests/${BRANCH_NAME.replace("/", "%2F")}",
                        parameters: [
                            string(name: 'GIT_COMMIT_TO_USE', value: env.GIT_COMMIT),
                            string(name: 'WILDCARD_DNS_DOMAIN', value: params.WILDCARD_DNS_DOMAIN)
                        ], wait: true
                }
            }
        }
    }

    post {
        always {
            script {
                if ( fileExists(env.TESTS_EXECUTED_FILE) ) {
                    dumpVerrazzanoSystemPods()
                    dumpCattleSystemPods()
                    dumpNginxIngressControllerLogs()
                    dumpVerrazzanoPlatformOperatorLogs()
                    dumpVerrazzanoApplicationOperatorLogs()
                    dumpOamKubernetesRuntimeLogs()
                    dumpVerrazzanoApiLogs()
                }
            }
            archiveArtifacts artifacts: '**/coverage.html,**/logs/**,**/verrazzano_images.txt,**/*-cluster-dump/**', allowEmptyArchive: true
            junit testResults: '**/*test-result.xml', allowEmptyResults: true

            sh """
                cd ${GO_REPO_PATH}/verrazzano/platform-operator
                make delete-cluster
                if [ -f ${POST_DUMP_FAILED_FILE} ]; then
                  echo "Failures seen during dumping of artifacts, treat post as failed"
                  exit 1
                fi
            """
        }
        failure {
            sh """
                curl -k -u ${JENKINS_READ_USR}:${JENKINS_READ_PSW} -o ${WORKSPACE}/build-console-output.log ${BUILD_URL}consoleText
            """
            archiveArtifacts artifacts: '**/build-console-output.log', allowEmptyArchive: true
            sh """
                curl -k -u ${JENKINS_READ_USR}:${JENKINS_READ_PSW} -o archive.zip ${BUILD_URL}artifact/*zip*/archive.zip
                oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_ARTIFACT_BUCKET} --name ${env.BRANCH_NAME}-${env.BUILD_NUMBER}/archive.zip --file archive.zip
                rm archive.zip
            """
            mail to: "${env.BUILD_NOTIFICATION_TO_EMAIL}", from: "${env.BUILD_NOTIFICATION_FROM_EMAIL}",
            subject: "Verrazzano: ${env.JOB_NAME} - Failed",
            body: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}\n\nSuspects:\n${SUSPECT_LIST}"
            script {
                if (env.JOB_NAME == "verrazzano/master" || env.JOB_NAME == "verrazzano/develop") {
                    pagerduty(resolve: false, serviceKey: "$SERVICE_KEY", incDescription: "Verrazzano: ${env.JOB_NAME} - Failed", incDetails: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}")
                    slackSend ( channel: "$SLACK_ALERT_CHANNEL", message: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}\n\nSuspects:\n${SUSPECT_LIST}" )
                }
            }
        }
        success {
            sh """
                if [ "${params.GENERATE_TARBALL}" == "true" ]; then
                    mkdir ${WORKSPACE}/tar-files
                    chmod uog+w ${WORKSPACE}/tar-files
                    cp $WORKSPACE/generated-verrazzano-bom.json ${WORKSPACE}/tar-files/verrazzano-bom.json
                    cp tools/scripts/vz-registry-image-helper.sh ${WORKSPACE}/tar-files/vz-registry-image-helper.sh
                    cp tools/scripts/README.md ${WORKSPACE}/tar-files/README.md
                    mkdir -p ${WORKSPACE}/tar-files/charts
                    cp  -r platform-operator/helm_config/charts/verrazzano-platform-operator ${WORKSPACE}/tar-files/charts
                    tools/scripts/generate_tarball.sh ${WORKSPACE}/tar-files/verrazzano-bom.json ${WORKSPACE}/tar-files ${WORKSPACE}/tarball.tar.gz
                    echo "git-commit=${env.GIT_COMMIT}" > $WORKSPACE/tarball-commit.txt
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/tarball-commit.txt --file $WORKSPACE/tarball-commit.txt
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/tarball.tar.gz --file ${WORKSPACE}/tarball.tar.gz
                fi
            """

            // If this is master and it was clean, record the commit in object store so the periodic test jobs can run against that rather than the head of master
            sh """
                if [ "${env.JOB_NAME}" == "verrazzano/master" ]; then
                    cd ${GO_REPO_PATH}/verrazzano
                    echo "git-commit=${env.GIT_COMMIT}" > $WORKSPACE/last-stable-commit.txt
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/last-stable-commit.txt --file $WORKSPACE/last-stable-commit.txt
                fi
            """
        }
        cleanup {
            deleteDir()
        }
    }
}

def runGinkgoRandomize(testSuitePath) {
    catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
        sh """
            cd ${GO_REPO_PATH}/verrazzano/tests/e2e
            ginkgo -p --randomizeAllSpecs -v -keepGoing --noColor ${testSuitePath}/...
        """
    }
}

def runGinkgo(testSuitePath) {
    catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
        sh """
            cd ${GO_REPO_PATH}/verrazzano/tests/e2e
            ginkgo -v -keepGoing --noColor ${testSuitePath}/...
        """
    }
}

def dumpK8sCluster(dumpDirectory) {
    sh """
        ${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d ${dumpDirectory} -r ${dumpDirectory}/analysis.report
    """
}

def dumpVerrazzanoSystemPods() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/platform-operator
        export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-system-pods.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -m "verrazzano system pods" || echo "failed" > ${POST_DUMP_FAILED_FILE}
        export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-system-certs.log"
        ./scripts/install/k8s-dump-objects.sh -o cert -n verrazzano-system -m "verrazzano system certs" || echo "failed" > ${POST_DUMP_FAILED_FILE}
        export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-system-kibana.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "vmi-system-kibana-*" -m "verrazzano system kibana log" -l -c kibana || echo "failed" > ${POST_DUMP_FAILED_FILE}
        export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-system-es-master.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "vmi-system-es-master-*" -m "verrazzano system kibana log" -l -c es-master || echo "failed" > ${POST_DUMP_FAILED_FILE}
    """
}

def dumpCattleSystemPods() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/platform-operator
        export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/cattle-system-pods.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n cattle-system -m "cattle system pods" || echo "failed" > ${POST_DUMP_FAILED_FILE}
        export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/rancher.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n cattle-system -r "rancher-*" -m "Rancher logs" -c rancher -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
    """
}

def dumpNginxIngressControllerLogs() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/platform-operator
        export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/nginx-ingress-controller.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n ingress-nginx -r "nginx-ingress-controller-*" -m "Nginx Ingress Controller" -c controller -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
    """
}

def dumpVerrazzanoPlatformOperatorLogs() {
    sh """
        ## dump out verrazzano-platform-operator logs
        mkdir -p ${WORKSPACE}/verrazzano-platform-operator/logs
        kubectl -n verrazzano-install logs --selector=app=verrazzano-platform-operator > ${WORKSPACE}/verrazzano-platform-operator/logs/verrazzano-platform-operator-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
        kubectl -n verrazzano-install describe pod --selector=app=verrazzano-platform-operator > ${WORKSPACE}/verrazzano-platform-operator/logs/verrazzano-platform-operator-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
        echo "verrazzano-platform-operator logs dumped to verrazzano-platform-operator-pod.log"
        echo "verrazzano-platform-operator pod description dumped to verrazzano-platform-operator-pod.out"
        echo "------------------------------------------"
    """
}

def dumpVerrazzanoApplicationOperatorLogs() {
    sh """
        ## dump out verrazzano-application-operator logs
        mkdir -p ${WORKSPACE}/verrazzano-application-operator/logs
        kubectl -n verrazzano-system logs --selector=app=verrazzano-application-operator > ${WORKSPACE}/verrazzano-application-operator/logs/verrazzano-application-operator-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
        kubectl -n verrazzano-system describe pod --selector=app=verrazzano-application-operator > ${WORKSPACE}/verrazzano-application-operator/logs/verrazzano-application-operator-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
        echo "verrazzano-application-operator logs dumped to verrazzano-application-operator-pod.log"
        echo "verrazzano-application-operator pod description dumped to verrazzano-application-operator-pod.out"
        echo "------------------------------------------"
    """
}

def dumpOamKubernetesRuntimeLogs() {
    sh """
        ## dump out oam-kubernetes-runtime logs
        mkdir -p ${WORKSPACE}/oam-kubernetes-runtime/logs
        kubectl -n verrazzano-system logs --selector=app.kubernetes.io/instance=oam-kubernetes-runtime > ${WORKSPACE}/oam-kubernetes-runtime/logs/oam-kubernetes-runtime-pod.log --tail -1 || echo "failed" > ${POST_DUMP_FAILED_FILE}
        kubectl -n verrazzano-system describe pod --selector=app.kubernetes.io/instance=oam-kubernetes-runtime > ${WORKSPACE}/verrazzano-application-operator/logs/oam-kubernetes-runtime-pod.out || echo "failed" > ${POST_DUMP_FAILED_FILE}
        echo "verrazzano-application-operator logs dumped to oam-kubernetes-runtime-pod.log"
        echo "verrazzano-application-operator pod description dumped to oam-kubernetes-runtime-pod.out"
        echo "------------------------------------------"
    """
}

def dumpVerrazzanoApiLogs() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/platform-operator
        export DIAGNOSTIC_LOG="${VERRAZZANO_INSTALL_LOGS_DIR}/verrazzano-api.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "verrazzano-api-*" -m "verrazzano api" -c verrazzano-api -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
    """
}

@NonCPS
def getCommitList() {
    echo "Checking for change sets"
    def commitList = []
    def changeSets = currentBuild.changeSets
    for (int i = 0; i < changeSets.size(); i++) {
        echo "get commits from change set"
        def commits = changeSets[i].items
        for (int j = 0; j < commits.length; j++) {
            def commit = commits[j]
            def id = commit.commitId
            echo "Add commit id: ${id}"
            commitList.add(id)
        }
    }
    return commitList
}

def trimIfGithubNoreplyUser(userIn) {
    if (userIn == null) {
        echo "Not a github noreply user, not trimming: ${userIn}"
        return userIn
    }
    if (userIn.matches(".*\\+.*@users.noreply.github.com.*")) {
        def userOut = userIn.substring(userIn.indexOf("+") + 1, userIn.indexOf("@"))
        return userOut;
    }
    if (userIn.matches(".*<.*@users.noreply.github.com.*")) {
        def userOut = userIn.substring(userIn.indexOf("<") + 1, userIn.indexOf("@"))
        return userOut;
    }
    if (userIn.matches(".*@users.noreply.github.com")) {
        def userOut = userIn.substring(0, userIn.indexOf("@"))
        return userOut;
    }
    echo "Not a github noreply user, not trimming: ${userIn}"
    return userIn
}

def getSuspectList(commitList, userMappings) {
    def retValue = ""
    if (commitList == null || commitList.size() == 0) {
        echo "No commits to form suspect list"
        return retValue
    }
    def suspectList = []
    for (int i = 0; i < commitList.size(); i++) {
        def id = commitList[i]
        def gitAuthor = sh(
            script: "git log --format='%ae' '$id^!'",
            returnStdout: true
        ).trim()
        if (gitAuthor != null) {
            def author = trimIfGithubNoreplyUser(gitAuthor)
            echo "DEBUG: author: ${gitAuthor}, ${author}, id: ${id}"
            if (userMappings.containsKey(author)) {
                def slackUser = userMappings.get(author)
                if (!suspectList.contains(slackUser)) {
                    echo "Added ${slackUser} as suspect"
                    retValue += " ${slackUser}"
                    suspectList.add(slackUser)
                }
            } else {
                // If we don't have a name mapping use the commit.author, at least we can easily tell if the mapping gets dated
                if (!suspectList.contains(author)) {
                    echo "Added ${author} as suspect"
                    retValue += " ${author}"
                   suspectList.add(author)
                }
            }
        } else {
            echo "No author returned from git"
        }
    }
    echo "returning suspect list: ${retValue}"
    return retValue
}
