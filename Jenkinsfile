// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

def DOCKER_IMAGE_TAG
def SKIP_ACCEPTANCE_TESTS = false

pipeline {
    options {
        skipDefaultCheckout true
    }

    agent {
       docker {
            image "${RUNNER_DOCKER_IMAGE}"
            args "${EXPERIMENTAL_RUNNER_DOCKER_ARGS}"
            registryUrl "${RUNNER_DOCKER_REGISTRY_URL}"
            registryCredentialsId 'ocir-pull-and-push-account'
            label "largeexperimental"
        }
    }

    parameters {
        booleanParam (description: 'Whether to kick off acceptance test run at the end of this build', name: 'RUN_ACCEPTANCE_TESTS', defaultValue: true)
        booleanParam (description: 'Whether to include the slow tests in the acceptance tests', name: 'RUN_SLOW_TESTS', defaultValue: false)
        booleanParam (description: 'Whether to dump k8s cluster on success (off by default can be useful to capture for comparing to failed cluster)', name: 'DUMP_K8S_CLUSTER_ON_SUCCESS', defaultValue: false)
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

        WEBLOGIC_PSW = credentials('weblogic-example-domain-password') // Needed by ToDoList example test
        DATABASE_PSW = credentials('todo-mysql-password') // Needed by ToDoList example test

        JENKINS_READ = credentials('jenkins-auditor')
    }

    stages {
        stage('Clean workspace and checkout') {
            steps {
                sh """
                    echo "${NODE_LABELS}"
                """

                script {
                    checkout scm
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
                    SHORT_COMMIT_HASH = sh(returnStdout: true, script: "git rev-parse --short HEAD").trim()
                    DOCKER_IMAGE_TAG = "${VERRAZZANO_DEV_VERSION}-${TIMESTAMP}-${SHORT_COMMIT_HASH}"
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
                success {
                    archiveArtifacts artifacts: "generated-operator.yaml", allowEmptyArchive: true
                }
            }
        }

        stage('Update operator.yaml') {
            when {
                allOf {
                    not { buildingTag() }
                }
            }
            environment {
                OCI_CLI_AUTH="instance_principal"
                OCI_OS_NAMESPACE = credentials('oci-os-namespace')
                OCI_OS_BUCKET="verrazzano-builds"
            }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/operator.yaml --file $WORKSPACE/generated-operator.yaml
                    oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SHORT_COMMIT_HASH}/operator.yaml --file $WORKSPACE/generated-operator.yaml
                   """
            }
        }

        stage('Build') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano
                    make docker-push VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} VERRAZZANO_ANALYSIS_IMAGE_NAME=${DOCKER_ANALYSIS_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
                   """
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
        }

        stage('Unit Tests') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano
                    make -B coverage
                    cp coverage.html ${WORKSPACE}
                    cp coverage.xml ${WORKSPACE}
                    build/copy-junit-output.sh ${WORKSPACE}
                """
            }
            post {
                always {
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
                always {
                    archiveArtifacts artifacts: '**/scanning-report.json', allowEmptyArchive: true
                }
            }
        }

        stage('Integration Tests') {
            when { not { buildingTag() } }
            steps {
                sh """
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
                """
            }
            post {
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
                        OCI_CLI_AUTH="instance_principal"
                        OCI_OS_NAMESPACE = credentials('oci-os-namespace')
                        OCI_OS_BUCKET="verrazzano-builds"
                    }
                    steps {
                        sh """
                            echo "tests will execute" > ${TESTS_EXECUTED_FILE}
                            echo "Create Kind cluster"
                            cd ${GO_REPO_PATH}/verrazzano/platform-operator
                            make cleanup-cluster
                            make create-cluster KIND_CONFIG="kind-config-ci.yaml"
                            ../ci/scripts/setup_kind_for_jenkins.sh

                            echo "Install metallb"
                            cd ${GO_REPO_PATH}/verrazzano
                            ./tests/e2e/config/scripts/install-metallb.sh

                            echo "Create Image Pull Secrets"
                            cd ${GO_REPO_PATH}/verrazzano
                            ./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}"
                            ./tests/e2e/config/scripts/create-image-pull-secret.sh github-packages "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}"
                            ./tests/e2e/config/scripts/create-image-pull-secret.sh ocr "${OCR_REPO}" "${OCR_CREDS_USR}" "${OCR_CREDS_PSW}"

                            echo "Install Platform Operator"
                            cd ${GO_REPO_PATH}/verrazzano

                            # Configure the deployment file to use an image pull secret for branches that have private images
                            oci --region us-phoenix-1 os object get --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${SHORT_COMMIT_HASH}/operator.yaml --file ${WORKSPACE}/downloaded-operator.yaml
                            cp ${WORKSPACE}/downloaded-operator.yaml ${WORKSPACE}/acceptance-test-operator.yaml

                            # Install the verrazzano-platform-operator
                            kubectl apply -f $WORKSPACE/acceptance-test-operator.yaml

                            # make sure ns exists
                            ./tests/e2e/config/scripts/check_verrazzano_ns_exists.sh verrazzano-install

                            # create secret in verrazzano-install ns
                            ./tests/e2e/config/scripts/create-image-pull-secret.sh "${IMAGE_PULL_SECRET}" "${DOCKER_REPO}" "${DOCKER_CREDS_USR}" "${DOCKER_CREDS_PSW}" "verrazzano-install"

                            # Configure the custom resource to install verrazzano on Kind
                            echo "Installing yq"
                            GO111MODULE=on go get github.com/mikefarah/yq/v4
                            export PATH=${HOME}/go/bin:${PATH}
                            ./tests/e2e/config/scripts/process_kind_install_yaml.sh ${INSTALL_CONFIG_FILE_KIND}

                            echo "Wait for Operator to be ready"
                            cd ${GO_REPO_PATH}/verrazzano
                            kubectl -n verrazzano-install rollout status deployment/verrazzano-platform-operator

                            echo "Installing Verrazzano on Kind"
                            kubectl apply -f ${INSTALL_CONFIG_FILE_KIND}

                            # wait for Verrazzano install to complete
                            ./tests/e2e/config/scripts/wait-for-verrazzano-install.sh
                        """
                    }
                    post {
                        always {
                            archiveArtifacts artifacts: "acceptance-test-operator.yaml,downloaded-operator.yaml", allowEmptyArchive: true
                            sh """
                                ## dump out install logs
                                mkdir -p ${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs
                                kubectl logs --selector=job-name=verrazzano-install-my-verrazzano > ${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/verrazzano-install.log --tail -1
                                kubectl describe pod --selector=job-name=verrazzano-install-my-verrazzano > ${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/verrazzano-install-job-pod.out
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
                            steps {
                                runGinkgoRandomize('verify-install')
                            }
                        }
                        stage('verify-infra restapi') {
                            steps {
                                runGinkgoRandomize('verify-infra/restapi')
                            }
                        }
                        stage('verify-infra oam') {
                            steps {
                                runGinkgoRandomize('verify-infra/oam')
                            }
                        }
                        stage('verify-infra vmi') {
                            steps {
                                runGinkgoRandomize('verify-infra/vmi')
                            }
                        }
                        stage('examples authorization policy') {
                            steps {
                                runGinkgo('istio/authz')
                            }
                        }
                        stage('examples logging helidon') {
                            steps {
                                runGinkgo('logging/helidon')
                            }
                        }
                        stage('examples todo') {
                            steps {
                                runGinkgo('examples/todo')
                            }
                        }
                        stage('examples socks') {
                            steps {
                                runGinkgo('examples/socks')
                            }
                        }
                        stage('examples spring') {
                            steps {
                                runGinkgo('examples/springboot')
                            }
                        }
                        stage('examples helidon') {
                            steps {
                                runGinkgo('examples/helidon')
                            }
                        }
                        stage('examples bobs') {
                            when {
                                expression {params.RUN_SLOW_TESTS == true}
                            }
                            steps {
                                runGinkgo('examples/bobsbooks')
                            }
                        }
                        stage('console ingress') {
                            steps {
                                runGinkgo('ingress/console')
                            }
                        }
                    }
                    post {
                        always {
                            archiveArtifacts artifacts: '**/coverage.html,**/logs/*', allowEmptyArchive: true
                            junit testResults: '**/*test-result.xml', allowEmptyResults: true
                        }
                    }
                }
            }

            post {
                failure {
                    script {
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
            mail to: "${env.BUILD_NOTIFICATION_TO_EMAIL}", from: "${env.BUILD_NOTIFICATION_FROM_EMAIL}",
            subject: "Verrazzano: ${env.JOB_NAME} - Failed",
            body: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}"
            script {
                if (env.JOB_NAME == "verrazzano/master" || env.JOB_NAME == "verrazzano/develop") {
                    pagerduty(resolve: false, serviceKey: "$SERVICE_KEY", incDescription: "Verrazzano: ${env.JOB_NAME} - Failed", incDetails: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}")
                    slackSend ( message: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}" )
                }
            }
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
        export DIAGNOSTIC_LOG="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/verrazzano-system-pods.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -m "verrazzano system pods" || echo "failed" > ${POST_DUMP_FAILED_FILE}
        export DIAGNOSTIC_LOG="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/verrazzano-system-certs.log"
        ./scripts/install/k8s-dump-objects.sh -o cert -n verrazzano-system -m "verrazzano system certs" || echo "failed" > ${POST_DUMP_FAILED_FILE}
        export DIAGNOSTIC_LOG="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/verrazzano-system-kibana.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "vmi-system-kibana-*" -m "verrazzano system kibana log" -l -c kibana || echo "failed" > ${POST_DUMP_FAILED_FILE}
        export DIAGNOSTIC_LOG="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/verrazzano-system-es-master.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "vmi-system-es-master-*" -m "verrazzano system kibana log" -l -c es-master || echo "failed" > ${POST_DUMP_FAILED_FILE}
    """
}

def dumpCattleSystemPods() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/platform-operator
        export DIAGNOSTIC_LOG="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/cattle-system-pods.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n cattle-system -m "cattle system pods" || echo "failed" > ${POST_DUMP_FAILED_FILE}
        export DIAGNOSTIC_LOG="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/rancher.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n cattle-system -r "rancher-*" -m "Rancher logs" -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
    """
}

def dumpNginxIngressControllerLogs() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/platform-operator
        export DIAGNOSTIC_LOG="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/nginx-ingress-controller.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n ingress-nginx -r "nginx-ingress-controller-*" -m "Nginx Ingress Controller" -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
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
        export DIAGNOSTIC_LOG="${WORKSPACE}/verrazzano/platform-operator/scripts/install/build/logs/verrazzano-api.log"
        ./scripts/install/k8s-dump-objects.sh -o pods -n verrazzano-system -r "verrazzano-api-*" -m "verrazzano api" -l || echo "failed" > ${POST_DUMP_FAILED_FILE}
    """
}
