// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

def DOCKER_IMAGE_TAG
def SKIP_ACCEPTANCE_TESTS = false
def SKIP_TRIGGERED_TESTS = false
def SUSPECT_LIST = ""
def VERRAZZANO_DEV_VERSION = ""
def tarfilePrefix=""
def storeLocation=""

def agentLabel = env.JOB_NAME.contains('master') ? "phx-large" : "large"

pipeline {
    options {
        skipDefaultCheckout true
        copyArtifactPermission('*');
        timestamps ()
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
        booleanParam (description: 'Whether to push images to OCIR', name: 'PUSH_TO_OCIR', defaultValue: false)
        booleanParam (description: 'Whether to fail the Integration Tests to test failure handling', name: 'SIMULATE_FAILURE', defaultValue: false)
        booleanParam (description: 'Whether to perform a scan of the built images', name: 'PERFORM_SCAN', defaultValue: false)
        booleanParam (description: 'Whether to wait for triggered tests or not. This defaults to false, this setting is useful for things like release automation that require everything to complete successfully', name: 'WAIT_FOR_TRIGGERED', defaultValue: false)
        choice (name: 'WILDCARD_DNS_DOMAIN',
                description: 'Wildcard DNS Domain',
                // 1st choice is the default value
                choices: [ "nip.io", "sslip.io"])
        choice (name: 'CRD_API_VERSION',
                description: 'This is the API crd version.',
                // 1st choice is the default value
                choices: [ "v1beta1", "v1alpha1"])
        string (name: 'CONSOLE_REPO_BRANCH',
                defaultValue: '',
                description: 'The branch to check out after cloning the console repository.',
                trim: true)
        booleanParam (description: 'Whether to emit metrics from the pipeline', name: 'EMIT_METRICS', defaultValue: true)
    }

    environment {
        TEST_ENV = "JENKINS"
        CLEAN_BRANCH_NAME = "${env.BRANCH_NAME.replace("/", "%2F")}"
        IS_PERIODIC_PIPELINE = "false"

        DOCKER_PLATFORM_CI_IMAGE_NAME = 'verrazzano-platform-operator-jenkins'
        DOCKER_PLATFORM_PUBLISH_IMAGE_NAME = 'verrazzano-platform-operator'
        DOCKER_PLATFORM_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_PLATFORM_PUBLISH_IMAGE_NAME : env.DOCKER_PLATFORM_CI_IMAGE_NAME}"
        DOCKER_APP_CI_IMAGE_NAME = 'verrazzano-application-operator-jenkins'
        DOCKER_APP_PUBLISH_IMAGE_NAME = 'verrazzano-application-operator'
        DOCKER_APP_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_APP_PUBLISH_IMAGE_NAME : env.DOCKER_APP_CI_IMAGE_NAME}"
        DOCKER_CLUSTER_CI_IMAGE_NAME = 'verrazzano-cluster-operator-jenkins'
        DOCKER_CLUSTER_PUBLISH_IMAGE_NAME = 'verrazzano-cluster-operator'
        DOCKER_CLUSTER_IMAGE_NAME = "${env.BRANCH_NAME ==~ /^release-.*/ || env.BRANCH_NAME == 'master' ? env.DOCKER_CLUSTER_PUBLISH_IMAGE_NAME : env.DOCKER_CLUSTER_CI_IMAGE_NAME}"
        CREATE_LATEST_TAG = "${env.BRANCH_NAME == 'master' ? '1' : '0'}"
        USE_V8O_DOC_STAGE = "${env.BRANCH_NAME ==~ /^release-.*/ ? 'false' : 'true'}"
        GOPATH = '/home/opc/go'
        GO_REPO_PATH = "${GOPATH}/src/github.com/verrazzano"
        DOCKER_CREDS = credentials('github-packages-credentials-rw')
        DOCKER_EMAIL = credentials('github-packages-email')
        DOCKER_REPO = 'ghcr.io'
        DOCKER_NAMESPACE = 'verrazzano'
        NETRC_FILE = credentials('netrc')
        GITHUB_PKGS_CREDS = credentials('github-packages-credentials-rw')
        SERVICE_KEY = credentials('PAGERDUTY_SERVICE_KEY')

        CLUSTER_NAME = 'verrazzano'
        POST_DUMP_FAILED_FILE = "${WORKSPACE}/post_dump_failed_file.tmp"
        TESTS_EXECUTED_FILE = "${WORKSPACE}/tests_executed_file.tmp"
        KUBECONFIG = "${WORKSPACE}/test_kubeconfig"
        VERRAZZANO_KUBECONFIG = "${KUBECONFIG}"
        OCR_CREDS = credentials('ocr-pull-and-push-account')
        OCR_REPO = 'container-registry.oracle.com'
        IMAGE_PULL_SECRET = 'verrazzano-container-registry'
        INSTALL_CONFIG_FILE_KIND = "./tests/e2e/config/scripts/${params.CRD_API_VERSION}/install-verrazzano-kind.yaml"
        INSTALL_PROFILE = "dev"
        VZ_ENVIRONMENT_NAME = "default"
        TEST_SCRIPTS_DIR = "${GO_REPO_PATH}/verrazzano/tests/e2e/config/scripts"

        WEBLOGIC_PSW = credentials('weblogic-example-domain-password') // Needed by ToDoList example test
        DATABASE_PSW = credentials('todo-mysql-password') // Needed by ToDoList example test

        // used for console artifact capture on failure
        JENKINS_READ = credentials('jenkins-auditor')

        OCI_CLI_AUTH="instance_principal"
        OCI_OS_NAMESPACE = credentials('oci-os-namespace')
        OCI_OS_ARTIFACT_BUCKET="build-failure-artifacts"
        OCI_OS_BUCKET="verrazzano-builds"
        OCI_OS_COMMIT_BUCKET="verrazzano-builds-by-commit"
        OCI_OS_REGION="us-phoenix-1"

        // used to emit metrics
        PROMETHEUS_GW_URL = credentials('prometheus-dev-url')
        PROMETHEUS_CREDENTIALS = credentials('prometheus-credentials')

        OCIR_SCAN_COMPARTMENT = credentials('ocir-scan-compartment')
        OCIR_SCAN_TARGET = credentials('ocir-scan-target')
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
                moveContentToGoRepoPath()

                script {
                    def props = readProperties file: '.verrazzano-development-version'
                    VERRAZZANO_DEV_VERSION = props['verrazzano-development-version']
                    TIMESTAMP = sh(returnStdout: true, script: "date +%Y%m%d%H%M%S").trim()
                    SHORT_COMMIT_HASH = sh(returnStdout: true, script: "echo $env.GIT_COMMIT | head -c 8")
                    env.VERRAZZANO_VERSION = "${VERRAZZANO_DEV_VERSION}"
                    if (!"${env.GIT_BRANCH}".startsWith("release-")) {
                        env.VERRAZZANO_VERSION = "${env.VERRAZZANO_VERSION}-${env.BUILD_NUMBER}+${SHORT_COMMIT_HASH}"
                    }
                    DOCKER_IMAGE_TAG = "v${VERRAZZANO_DEV_VERSION}-${TIMESTAMP}-${SHORT_COMMIT_HASH}"
                    // update the description with some meaningful info
                    currentBuild.description = SHORT_COMMIT_HASH + " : " + env.GIT_COMMIT
                    def currentCommitHash = env.GIT_COMMIT
                    def commitList = getCommitList()
                    withCredentials([file(credentialsId: 'jenkins-to-slack-users', variable: 'JENKINS_TO_SLACK_JSON')]) {
                        def userMappings = readJSON file: JENKINS_TO_SLACK_JSON
                        SUSPECT_LIST = getSuspectList(commitList, userMappings)
                        echo "Suspect list: ${SUSPECT_LIST}"
                    }

                    setEffectiveDocsVersion()
                }
            }
        }

    }

    post {
        always {
            archiveArtifacts artifacts: '**/coverage.html,**/logs/**,**/verrazzano_images.txt,**/*cluster-snapshot*/**', allowEmptyArchive: true
            junit testResults: '**/*test-result.xml', allowEmptyResults: true
        }
        failure {
            sh """
                curl -k -u ${JENKINS_READ_USR}:${JENKINS_READ_PSW} -o ${WORKSPACE}/build-console-output.log ${BUILD_URL}consoleText
            """
            archiveArtifacts artifacts: '**/build-console-output.log', allowEmptyArchive: true
            sh """
                curl -k -u ${JENKINS_READ_USR}:${JENKINS_READ_PSW} -o archive.zip ${BUILD_URL}artifact/*zip*/archive.zip
                oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_ARTIFACT_BUCKET} --name ${env.JOB_NAME}/${env.BRANCH_NAME}/${env.BUILD_NUMBER}/archive.zip --file archive.zip
                rm archive.zip
            """
            script {
                if (isPagerDutyEnabled() && (env.JOB_NAME == "verrazzano/master" || env.JOB_NAME ==~ "verrazzano/release-1.*")) {
                    pagerduty(resolve: false, serviceKey: "$SERVICE_KEY", incDescription: "Verrazzano: ${env.JOB_NAME} - Failed", incDetails: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}")
                }
                if (env.JOB_NAME == "verrazzano/master" || env.JOB_NAME ==~ "verrazzano/release-1.*" || env.BRANCH_NAME ==~ "mark/*") {
                    slackSend ( channel: "$SLACK_ALERT_CHANNEL", message: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}\n\nSuspects:\n${SUSPECT_LIST}" )
                }
            }
        }
        cleanup {
            metricBuildDuration()
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

// Called in Stage CLI steps
def buildVerrazzanoCLI(dockerImageTag) {
    sh """
        cd ${GO_REPO_PATH}/verrazzano/tools/vz
        make go-build DOCKER_IMAGE_TAG=${dockerImageTag}
        ${GO_REPO_PATH}/verrazzano/ci/scripts/save_tooling.sh ${env.BRANCH_NAME} ${SHORT_COMMIT_HASH}
        cp out/linux_amd64/vz ${GO_REPO_PATH}/vz
    """
}

def checkRepoClean() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano
        echo 'Check for forgotten manifest/generate actions...'
        (cd platform-operator; make check-repo-clean)
        (cd application-operator; make check-repo-clean)
    """
}

// Called in Stage Build steps
// Makes target docker push for application/platform operator and analysis
def buildImages(dockerImageTag) {
    sh """
        cd ${GO_REPO_PATH}/verrazzano
        echo 'Now build...'
        make docker-push VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME=${DOCKER_APP_IMAGE_NAME} VERRAZZANO_CLUSTER_OPERATOR_IMAGE_NAME=${DOCKER_CLUSTER_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${dockerImageTag} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
        cp ${GO_REPO_PATH}/verrazzano/platform-operator/out/generated-verrazzano-bom.json $WORKSPACE/generated-verrazzano-bom.json
        cp $WORKSPACE/generated-verrazzano-bom.json $WORKSPACE/verrazzano-bom.json
        ${GO_REPO_PATH}/verrazzano/tools/scripts/generate_image_list.sh $WORKSPACE/generated-verrazzano-bom.json $WORKSPACE/verrazzano_images.txt
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
        DOCKER_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME=${DOCKER_APP_IMAGE_NAME} VERRAZZANO_CLUSTER_OPERATOR_IMAGE_NAME=${DOCKER_CLUSTER_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${dockerImageTag} OPERATOR_YAML=$WORKSPACE/generated-operator.yaml make generate-operator-yaml
    """
}

// Called in Stage Save Generated Files steps
def saveGeneratedFiles() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/operator.yaml --file $WORKSPACE/generated-operator.yaml
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/operator.yaml --file $WORKSPACE/generated-operator.yaml
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/generated-verrazzano-bom.json --file $WORKSPACE/generated-verrazzano-bom.json
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/generated-verrazzano-bom.json --file $WORKSPACE/generated-verrazzano-bom.json
    """
}

def saveCLIExecutable() {
    sh """
        cd ${GO_REPO_PATH}/verrazzano
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/vz-linux-amd64.tar.gz --file $WORKSPACE/vz-linux-amd64.tar.gz
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/vz-linux-amd64.tar.gz --file $WORKSPACE/vz-linux-amd64.tar.gz
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_BUCKET} --name ${env.BRANCH_NAME}/vz-linux-amd64.tar.gz.sha256 --file $WORKSPACE/vz-linux-amd64.tar.gz.sha256
        oci --region us-phoenix-1 os object put --force --namespace ${OCI_OS_NAMESPACE} -bn ${OCI_OS_COMMIT_BUCKET} --name ephemeral/${env.BRANCH_NAME}/${SHORT_COMMIT_HASH}/vz-linux-amd64.tar.gz.sha256 --file $WORKSPACE/vz-linux-amd64.tar.gz.sha256
    """
}

// Called in many parallel stages of Stage Run Acceptance Tests
def runGinkgoRandomize(testSuitePath) {
    catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
        sh """
            cd ${GO_REPO_PATH}/verrazzano/tests/e2e
            ginkgo -p --randomize-all -v --keep-going --no-color ${testSuitePath}/...
            ../../build/copy-junit-output.sh ${WORKSPACE}
        """
    }
}

// Called in many parallel stages of Stage Run Acceptance Tests
def runGinkgo(testSuitePath) {
    catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
        sh """
            cd ${GO_REPO_PATH}/verrazzano/tests/e2e
            ginkgo -v --keep-going --no-color ${testSuitePath}/...
            ../../build/copy-junit-output.sh ${WORKSPACE}
        """
    }
}

// Called in Stage Acceptance Tests post
def dumpK8sCluster(dumpDirectory) {
    sh """
        mkdir -p ${dumpDirectory}/cluster-snapshot
        ${GO_REPO_PATH}/verrazzano/tools/scripts/k8s-dump-cluster.sh -d ${dumpDirectory}/cluster-snapshot -r ${dumpDirectory}/cluster-snapshot/analysis.report
        ${GO_REPO_PATH}/vz bug-report --report-file ${dumpDirectory}/bug-report.tar.gz
        mkdir -p ${dumpDirectory}/bug-report
        tar -xvf ${dumpDirectory}/bug-report.tar.gz -C ${dumpDirectory}/bug-report
    """
}

def setEffectiveDocsVersion() {
    sh """
        echo "USE_V8O_DOC_STAGE before is: $USE_V8O_DOC_STAGE"
        USE_V8O_DOC_STAGE=$VERRAZZANO_DEV_VERSION
        url="https://verrazzano.io/$USE_V8O_DOC_STAGE/docs/troubleshooting/diagnostictools/analysisadvice/"
        if curl $url --compressed | grep '404 Page not found'; then
          USE_V8O_DOC_STAGE="devel"
        fi
        echo "USE_V8O_DOC_STAGE after is: $USE_V8O_DOC_STAGE"
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
    def suspectList = []
    if (commitList == null || commitList.size() == 0) {
        echo "No commits to form suspect list"
    } else {
        for (int i = 0; i < commitList.size(); i++) {
            def id = commitList[i]
            try {
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
            } catch (Exception e) {
                echo "INFO: Problem processing commit ${id}, skipping commit: " + e.toString()
            }
        }
    }
    def startedByUser = "";
    def causes = currentBuild.getBuildCauses()
    echo "causes: " + causes.toString()
    for (cause in causes) {
        def causeString = cause.toString()
        echo "current cause: " + causeString
        def causeInfo = readJSON text: causeString
        if (causeInfo.userId != null) {
            startedByUser = causeInfo.userId
        }
    }

    if (startedByUser.length() > 0) {
        echo "Build was started by a user, adding them to the suspect notification list: ${startedByUser}"
        def author = trimIfGithubNoreplyUser(startedByUser)
        echo "DEBUG: author: ${startedByUser}, ${author}"
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
        echo "Build not started by a user, not adding to notification list"
    }
    echo "returning suspect list: ${retValue}"
    return retValue
}

def metricJobName(stageName) {
    job = env.JOB_NAME.split("/")[0]
    job = '_' + job.replaceAll('-','_')
    if (stageName) {
        job = job + '_' + stageName
    }
    return job
}

def metricTimerStart(metricName) {
    def timerStartName = "${metricName}_START"
    env."${timerStartName}" = sh(returnStdout: true, script: "date +%s").trim()
}

// Construct the set of labels/dimensions for the metrics
def getMetricLabels() {
    def buildNumber = String.format("%010d", env.BUILD_NUMBER.toInteger())
    labels = 'build_number=\\"' + "${buildNumber}"+'\\",' +
             'jenkins_build_number=\\"' + "${env.BUILD_NUMBER}"+'\\",' +
             'jenkins_job=\\"' + "${env.JOB_NAME}".replace("%2F","/") + '\\",' +
             'commit_sha=\\"' + "${env.GIT_COMMIT}"+'\\"'
    return labels
}

def metricTimerEnd(metricName, status) {
    def timerStartName = "${metricName}_START"
    def timerEndName   = "${metricName}_END"
    env."${timerEndName}" = sh(returnStdout: true, script: "date +%s").trim()
    if (params.EMIT_METRICS) {
        long x = env."${timerStartName}" as long;
        long y = env."${timerEndName}" as long;
        def dur = (y-x)
        labels = getMetricLabels()
        withCredentials([usernameColonPassword(credentialsId: 'prometheus-credentials', variable: 'PROMETHEUS_CREDENTIALS')]) {
            EMIT = sh(returnStdout: true, script: "ci/scripts/metric_emit.sh ${PROMETHEUS_GW_URL} ${PROMETHEUS_CREDENTIALS} ${metricName} ${env.GIT_BRANCH} $labels ${status} ${dur}")
            echo "emit prometheus metrics: $EMIT"
            return EMIT
        }
    } else {
        return ''
    }
}

// Emit the metrics indicating the duration and result of the build
def metricBuildDuration() {
    def status = "${currentBuild.currentResult}".trim()
    long duration = "${currentBuild.duration}" as long;
    long durationInSec = (duration/1000)
    testMetric = metricJobName('')
    def metricValue = "-1"
    statusLabel = status.substring(0,1)
    if (status.equals("SUCCESS")) {
        metricValue = "1"
    } else if (status.equals("FAILURE")) {
        metricValue = "0"
    } else {
        // Consider every other status as a single label
        statusLabel = "A"
    }
    if (params.EMIT_METRICS) {
        labels = getMetricLabels()
        labels = labels + ',result=\\"' + "${statusLabel}"+'\\"'
        withCredentials([usernameColonPassword(credentialsId: 'prometheus-credentials', variable: 'PROMETHEUS_CREDENTIALS')]) {
            METRIC_STATUS = sh(returnStdout: true, returnStatus: true, script: "ci/scripts/metric_emit.sh ${PROMETHEUS_GW_URL} ${PROMETHEUS_CREDENTIALS} ${testMetric}_job ${env.BRANCH_NAME} $labels ${metricValue} ${durationInSec}")
            echo "Publishing the metrics for build duration and status returned status code $METRIC_STATUS"
        }
    }
}
