// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

def DOCKER_IMAGE_TAG
def SKIP_ACCEPTANCE_TESTS = false
def SKIP_TRIGGERED_TESTS = false
def SUSPECT_LIST = ""
def SCAN_IMAGE_PATCH_OPERATOR = false
def VERRAZZANO_DEV_VERSION = ""
def tarfilePrefix=""
def storeLocation=""

def agentLabel = env.JOB_NAME.contains('master') ? "phxlarge" : "VM.Standard2.8_1_0"

pipeline {
    options {
        skipDefaultCheckout true
        copyArtifactPermission('*');
        timestamps ()
    }

    agent {
       docker {
            image "${RUNNER_DOCKER_IMAGE_1_0}"
            args "${RUNNER_DOCKER_ARGS_1_0}"
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
        booleanParam (description: 'Whether to generate the analysis tool', name: 'GENERATE_TOOL', defaultValue: false)
        booleanParam (description: 'Whether to generate a tarball', name: 'GENERATE_TARBALL', defaultValue: false)
        booleanParam (description: 'Whether to push images to OCIR', name: 'PUSH_TO_OCIR', defaultValue: false)
        booleanParam (description: 'Whether to fail the Integration Tests to test failure handling', name: 'SIMULATE_FAILURE', defaultValue: false)
        booleanParam (description: 'Whether to wait for triggered tests or not. This defaults to false, this setting is useful for things like release automation that require everything to complete successfully', name: 'WAIT_FOR_TRIGGERED', defaultValue: false)
        choice (name: 'WILDCARD_DNS_DOMAIN',
                description: 'Wildcard DNS Domain',
                // 1st choice is the default value
                choices: [ "nip.io", "sslip.io"])
        string (name: 'CONSOLE_REPO_BRANCH',
                defaultValue: 'master',
                description: 'The branch to check out after cloning the console repository.',
                trim: true)
    }

    environment {
        DOCKER_ANALYSIS_CI_IMAGE_NAME = 'verrazzano-analysis-jenkins'
        DOCKER_ANALYSIS_PUBLISH_IMAGE_NAME = 'verrazzano-analysis'
        DOCKER_ANALYSIS_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_ANALYSIS_PUBLISH_IMAGE_NAME : env.DOCKER_ANALYSIS_CI_IMAGE_NAME}"
        DOCKER_PLATFORM_CI_IMAGE_NAME = 'verrazzano-platform-operator-jenkins'
        DOCKER_PLATFORM_PUBLISH_IMAGE_NAME = 'verrazzano-platform-operator'
        DOCKER_PLATFORM_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_PLATFORM_PUBLISH_IMAGE_NAME : env.DOCKER_PLATFORM_CI_IMAGE_NAME}"
        DOCKER_IMAGE_PATCH_CI_IMAGE_NAME = 'verrazzano-image-patch-operator-jenkins'
        DOCKER_IMAGE_PATCH_PUBLISH_IMAGE_NAME = 'verrazzano-image-patch-operator'
        DOCKER_IMAGE_PATCH_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_IMAGE_PATCH_PUBLISH_IMAGE_NAME : env.DOCKER_IMAGE_PATCH_CI_IMAGE_NAME}"
        DOCKER_WIT_CI_IMAGE_NAME = 'verrazzano-weblogic-image-tool-jenkins'
        DOCKER_WIT_PUBLISH_IMAGE_NAME = 'verrazzano-weblogic-image-tool'
        DOCKER_WIT_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_WIT_PUBLISH_IMAGE_NAME : env.DOCKER_WIT_CI_IMAGE_NAME}"
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
        GITHUB_PKGS_CREDS = credentials('github-packages-credentials-rw')
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

        OCIR_SCAN_REGISTRY = credentials('ocir-scan-registry')
        OCIR_SCAN_REPOSITORY_PATH = credentials('ocir-scan-repository-path')
        DOCKER_SCAN_CREDS = credentials('v8odev-ocir')
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
                moveContentToGoRepoPath()

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
                        expression {params.GENERATE_TOOL == true};
                    }
                }
            }
            steps {
                buildAnalysisTool("${DOCKER_IMAGE_TAG}")
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                always {
                    archiveArtifacts artifacts: '**/*.tar.gz*', allowEmptyArchive: true
                }
            }
        }

        stage('Generate operator.yaml') {
            when { not { buildingTag() } }
            steps {
                generateOperatorYaml("${DOCKER_IMAGE_TAG}")
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
                buildImages("${DOCKER_IMAGE_TAG}")
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                success {
                    archiveArtifacts artifacts: "generated-verrazzano-bom.json,verrazzano_images.txt", allowEmptyArchive: true
                }
            }
        }

        stage('Image Patch Operator') {
            when {
                allOf {
                    not { buildingTag() }
                    changeset "image-patch-operator/**"
                }
            }
            steps {
                buildImagePatchOperator("${DOCKER_IMAGE_TAG}")
                buildWITImage("${DOCKER_IMAGE_TAG}")
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                success {
                    script {
                        SCAN_IMAGE_PATCH_OPERATOR = true
                    }
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
                saveGeneratedFiles()
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
                qualityCheck()
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
                    scanContainerImage "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_PLATFORM_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
                    scanContainerImage "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_OAM_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
                    scanContainerImage "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_ANALYSIS_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
                    if (SCAN_IMAGE_PATCH_OPERATOR == true) {
                        scanContainerImage "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_IMAGE_PATCH_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
                    }
                }
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                always {
                    archiveArtifacts artifacts: '**/scanning-report*.json', allowEmptyArchive: true
                }
            }
        }

        stage('Integration Tests') {
            when { not { buildingTag() } }
            steps {
                integrationTests("${DOCKER_IMAGE_TAG}")
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
                    }
                }
                always {
                    archiveArtifacts artifacts: '**/coverage.html,**/logs/*,**/*-cluster-dump/**,**/install.sh.log', allowEmptyArchive: true
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

        stage('Kind Acceptance Tests on 1.18') {
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

            steps {
                script {
                    build job: "verrazzano-new-kind-acceptance-tests/${BRANCH_NAME.replace("/", "%2F")}",
                        parameters: [
                            string(name: 'KUBERNETES_CLUSTER_VERSION', value: '1.18'),
                            string(name: 'GIT_COMMIT_TO_USE', value: env.GIT_COMMIT),
                            string(name: 'WILDCARD_DNS_DOMAIN', value: params.WILDCARD_DNS_DOMAIN),
                            booleanParam(name: 'RUN_SLOW_TESTS', value: params.RUN_SLOW_TESTS),
                            booleanParam(name: 'DUMP_K8S_CLUSTER_ON_SUCCESS', value: params.DUMP_K8S_CLUSTER_ON_SUCCESS),
                            booleanParam(name: 'CREATE_CLUSTER_USE_CALICO', value: params.CREATE_CLUSTER_USE_CALICO),
                            string(name: 'CONSOLE_REPO_BRANCH', value: params.CONSOLE_REPO_BRANCH)
                        ], wait: true
                }
            }
            post {
                failure {
                    script {
                        SKIP_TRIGGERED_TESTS = true
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
        stage('Zip Build and Test') {
            // If the tests are clean and this is a release branch or GENERATE_TARBALL == true,
            // generate the Verrazzano full product zip and run the Private Registry tests.
            // Optionally push images to OCIR for scanning.
            when {
                allOf {
                    not { buildingTag() }
                    expression {SKIP_TRIGGERED_TESTS == false}
                    anyOf {
                        expression{params.GENERATE_TARBALL == true};
                        expression{params.PUSH_TO_OCIR == true};
                    }
                }
            }
            stages{
                stage("Build Product Zip") {
                    steps {
                        script {
                            tarfilePrefix="verrazzano_${VERRAZZANO_DEV_VERSION}"
                            storeLocation="${env.BRANCH_NAME}/${tarfilePrefix}.zip"
                            generatedBOM="$WORKSPACE/generated-verrazzano-bom.json"
                            echo "Building zipfile, prefix: ${tarfilePrefix}, location:  ${storeLocation}"
                            sh """
                                ci/scripts/generate_product_zip.sh ${env.GIT_COMMIT} ${SHORT_COMMIT_HASH} ${env.BRANCH_NAME} ${tarfilePrefix} ${generatedBOM}
                            """
                        }
                    }
                }
                stage("Private Registry Test") {
                    steps {
                        script {
                            echo "Starting private registry test for ${storeLocation}, file prefix ${tarfilePrefix}"
                            build job: "verrazzano-private-registry/${BRANCH_NAME.replace("/", "%2F")}",
                                parameters: [
                                    string(name: 'GIT_COMMIT_TO_USE', value: env.GIT_COMMIT),
                                    string(name: 'WILDCARD_DNS_DOMAIN', value: params.WILDCARD_DNS_DOMAIN),
                                    string(name: 'ZIPFILE_LOCATION', value: storeLocation)
                                ], wait: true
                        }
                    }
                }
                stage("Push Images to OCIR") {
                    when {
                        expression{params.PUSH_TO_OCIR == true}
                    }
                    steps {
                        script {
                            try {
                                sh """
                                    echo "${DOCKER_SCAN_CREDS_PSW}" | docker login ${env.OCIR_SCAN_REGISTRY} -u ${DOCKER_SCAN_CREDS_USR} --password-stdin
                                """
                            } catch(error) {
                                echo "docker login failed, retrying after sleep"
                                retry(4) {
                                    sleep(30)
                                    sh """
                                    echo "${DOCKER_SCAN_CREDS_PSW}" | docker login ${env.OCIR_SCAN_REGISTRY} -u ${DOCKER_SCAN_CREDS_USR} --password-stdin
                                    """
                                }
                            }

                            sh """
                                echo "Pushing images to OCIR"
                                ci/scripts/push_to_ocir.sh
                            """
                        }
                    }
                }
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: '**/coverage.html,**/logs/**,**/verrazzano_images.txt,**/*-cluster-dump/**', allowEmptyArchive: true
            junit testResults: '**/*test-result.xml', allowEmptyResults: true
        }
        failure {
            sh """
                curl -k -u ${JENKINS_READ_USR}:${JENKINS_READ_PSW} -o ${WORKSPACE}/build-console-output.log ${BUILD_URL}consoleText
            """
            archiveArtifacts artifacts: '**/build-console-output.log', allowEmptyArchive: true
            sh """
                curl -k -u ${JENKINS_READ_USR}:${JENKINS_READ_PSW} -o archive.zip ${BUILD_URL}artifact/*zip*/archive.zip
                oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_ARTIFACT_BUCKET} --name ${env.BRANCH_NAME}/${env.BUILD_NUMBER}/archive.zip --file archive.zip
                rm archive.zip
            """
            script {
                if (isPagerDutyEnabled() && env.JOB_NAME == "verrazzano/master" || env.JOB_NAME ==~ "verrazzano/release-1.*") {
                    pagerduty(resolve: false, serviceKey: "$SERVICE_KEY", incDescription: "Verrazzano: ${env.JOB_NAME} - Failed", incDetails: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}")
                }
                if (env.JOB_NAME == "verrazzano/master" || env.JOB_NAME ==~ "verrazzano/release-*" || env.BRANCH_NAME ==~ "mark/*") {
                    slackSend ( channel: "$SLACK_ALERT_CHANNEL", message: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}\n\nSuspects:\n${SUSPECT_LIST}" )
                }
            }
        }
        success {
            storePipelineArtifacts()
        }
        cleanup {
            deleteDir()
        }
    }
}

def isPagerDutyEnabled() {
    // this controls whether PD alerts are enabled
    if (NOTIFY_PAGERDUTY_MAINJOB_FAILURES.equals("true")) {
        echo "Pager-Duty notifications enabled via global override setting"
        return true
    }
    return false
}

// Called in Stage Clean workspace and checkout steps
def moveContentToGoRepoPath() {
    sh """
        rm -rf ${GO_REPO_PATH}/verrazzano
        mkdir -p ${GO_REPO_PATH}/verrazzano
        tar cf - . | (cd ${GO_REPO_PATH}/verrazzano/ ; tar xf -)
    """
}

// Called in final post success block of pipeline
def storePipelineArtifacts() {
    script {
        // If this is master and it was clean, record the commit in object store so the periodic test jobs can run against that rather than the head of master
        sh """
            if [ "${env.JOB_NAME}" == "verrazzano/master" ]; then
                cd ${GO_REPO_PATH}/verrazzano
                echo "git-commit=${env.GIT_COMMIT}" > $WORKSPACE/last-stable-commit.txt
                oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name master/last-stable-commit.txt --file $WORKSPACE/last-stable-commit.txt
            fi
        """
    }
}

// Called in Stage Integration Tests steps
def integrationTests(dockerImageTag) {
    sh """
        if [ "${params.SIMULATE_FAILURE}" == "true" ]; then
            echo "Simulate failure from a stage"
            exit 1
        fi
        cd ${GO_REPO_PATH}/verrazzano/platform-operator

        make cleanup-cluster
        make create-cluster KIND_CONFIG="kind-config-ci.yaml"
        ../ci/scripts/setup_kind_for_jenkins.sh
        make integ-test CLUSTER_DUMP_LOCATION=${WORKSPACE}/platform-operator-integ-cluster-dump DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} DOCKER_IMAGE_TAG=${dockerImageTag}
        ../build/copy-junit-output.sh ${WORKSPACE}
        cd ${GO_REPO_PATH}/verrazzano/application-operator
        make cleanup-cluster
        make integ-test KIND_CONFIG="kind-config-ci.yaml" CLUSTER_DUMP_LOCATION=${WORKSPACE}/application-operator-integ-cluster-dump DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} DOCKER_IMAGE_TAG=${dockerImageTag}
        ../build/copy-junit-output.sh ${WORKSPACE}
        make cleanup-cluster
    """
}

// Called in Stage Analysis Tool steps
def buildAnalysisTool(dockerImageTag) {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/tools/analysis
        make go-build DOCKER_IMAGE_TAG=${dockerImageTag}
        ${GO_REPO_PATH}/verrazzano/ci/scripts/save_tooling.sh ${env.BRANCH_NAME} ${SHORT_COMMIT_HASH}
    """
}

// Called in Stage Build steps
// Makes target docker push for application/platform operator and analysis
def buildImages(dockerImageTag) {
    sh """
        cd ${GO_REPO_PATH}/verrazzano
        make docker-push VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} VERRAZZANO_ANALYSIS_IMAGE_NAME=${DOCKER_ANALYSIS_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${dockerImageTag} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
        cp ${GO_REPO_PATH}/verrazzano/platform-operator/out/generated-verrazzano-bom.json $WORKSPACE/generated-verrazzano-bom.json
        ${GO_REPO_PATH}/verrazzano/tools/scripts/generate_image_list.sh $WORKSPACE/generated-verrazzano-bom.json $WORKSPACE/verrazzano_images.txt
    """
}

// Called in Stage Image Patch Operator
// Makes target docker-push-ipo
def buildImagePatchOperator(dockerImageTag) {
    sh """
        cd ${GO_REPO_PATH}/verrazzano
        make docker-push-ipo VERRAZZANO_IMAGE_PATCH_OPERATOR_IMAGE_NAME=${DOCKER_IMAGE_PATCH_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${dockerImageTag} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
    """
}

// Called in Stage Image Patch Operator
def buildWITImage(dockerImageTag) {
    sh """
        cd ${GO_REPO_PATH}/verrazzano
        make docker-push-wit VERRAZZANO_WEBLOGIC_IMAGE_TOOL_IMAGE_NAME=${DOCKER_WIT_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${dockerImageTag} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
    """
}

// Called in Stage Generate operator.yaml steps
def generateOperatorYaml(dockerImageTag) {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/platform-operator
        case "${env.BRANCH_NAME}" in
            master|release-*)
                ;;
            *)
                echo "Adding image pull secrets to operator.yaml for non master/release branch"
                export IMAGE_PULL_SECRETS=verrazzano-container-registry
        esac
        DOCKER_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${dockerImageTag} OPERATOR_YAML=$WORKSPACE/generated-operator.yaml make generate-operator-yaml
    """
}

// Called in Stage Quality and Compliance Checks steps
// Makes target check to run all linters
def qualityCheck() {
    sh """
        echo "run all linters"
        cd ${GO_REPO_PATH}/verrazzano
        make check

        echo "copyright scan"
        time make copyright-check

        echo "Third party license check"
    """
}

// Called in Stage Save Generated Files steps
def saveGeneratedFiles() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/operator.yaml --file $WORKSPACE/generated-operator.yaml
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/operator.yaml --file $WORKSPACE/generated-operator.yaml
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/generated-verrazzano-bom.json --file $WORKSPACE/generated-verrazzano-bom.json
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/generated-verrazzano-bom.json --file $WORKSPACE/generated-verrazzano-bom.json
    """
}

// Called in many parallel stages of Stage Run Acceptance Tests
def runGinkgoRandomize(testSuitePath) {
    catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
        sh """
            cd ${GO_REPO_PATH}/verrazzano/tests/e2e
            ginkgo -p --randomizeAllSpecs -v -keepGoing --noColor ${testSuitePath}/...
            ../../build/copy-junit-output.sh ${WORKSPACE}
        """
    }
}

// Called in many parallel stages of Stage Run Acceptance Tests
def runGinkgo(testSuitePath) {
    catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
        sh """
            cd ${GO_REPO_PATH}/verrazzano/tests/e2e
            ginkgo -v -keepGoing --noColor ${testSuitePath}/...
            ../../build/copy-junit-output.sh ${WORKSPACE}
        """
    }
}

// Called in Stage Acceptance Tests post
def dumpK8sCluster(dumpDirectory) {
    sh """
        ${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d ${dumpDirectory} -r ${dumpDirectory}/analysis.report
    """
}


// Called in Stage Clean workspace and checkout steps
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
