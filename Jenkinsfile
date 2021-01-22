// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

def DOCKER_IMAGE_TAG

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
        }
    }

    parameters {
        string (name: 'RELEASE_VERSION',
                defaultValue: '',
                description: 'Release version used for the version of helm chart and tag for the image:\n'+
                'When RELEASE_VERSION is not defined, version will be determined by incrementing last minor release version by 1, for example:\n'+
                'When RELEASE_VERSION is v0.1.0, image tag will be v0.1.0 and helm chart version is also v0.1.0.\n'+
                'When RELEASE_VERSION is not specified and last release version is v0.1.0, image tag will be v0.1.1 and helm chart version is also v0.1.1.',
                trim: true)
        string (name: 'RELEASE_DESCRIPTION',
                defaultValue: '',
                description: 'Brief description for the release.',
                trim: true)
        string (name: 'RELEASE_BRANCH',
                defaultValue: 'master',
                description: 'Branch to create release from, change this to enable release from a non master branch, e.g.\n'+
                'When the branch being built is master then release will always be created when RELEASE_BRANCH has the default value - master.\n'+
                'When the branch being built is any non-master branch - release can be created by setting RELEASE_BRANCH to same value as non-master branch, else it is skipped.\n',
                trim: true)
        string (name: 'ACCEPTANCE_TESTS_BRANCH',
                defaultValue: 'master',
                description: 'Branch or tag of verrazzano acceptance tests, on which to kick off the tests',
                trim: true
        )
        booleanParam (description: 'Whether to kick off acceptance test run at the end of this build', name: 'RUN_ACCEPTANCE_TESTS', defaultValue: true)
    }

    environment {
        DOCKER_PLATFORM_CI_IMAGE_NAME = 'verrazzano-platform-operator-jenkins'
        DOCKER_PLATFORM_PUBLISH_IMAGE_NAME = 'verrazzano-platform-operator'
        DOCKER_PLATFORM_IMAGE_NAME = "${env.BRANCH_NAME == 'develop' || env.BRANCH_NAME == 'master' ? env.DOCKER_PLATFORM_PUBLISH_IMAGE_NAME : env.DOCKER_PLATFORM_CI_IMAGE_NAME}"
        DOCKER_OAM_CI_IMAGE_NAME = 'verrazzano-application-operator-jenkins'
        DOCKER_OAM_PUBLISH_IMAGE_NAME = 'verrazzano-application-operator'
        DOCKER_OAM_IMAGE_NAME = "${env.BRANCH_NAME == 'develop' || env.BRANCH_NAME == 'master' ? env.DOCKER_OAM_PUBLISH_IMAGE_NAME : env.DOCKER_OAM_CI_IMAGE_NAME}"
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
        SKIP_ACCEPTANCE_TESTS = "false"
    }

    stages {
        stage('Clean workspace and checkout') {
            steps {
                script {
                    checkout scm
                }
                sh """
                    cp -f "${NETRC_FILE}" $HOME/.netrc
                    chmod 600 $HOME/.netrc
                """

                sh """
                    echo "${DOCKER_CREDS_PSW}" | docker login ${env.DOCKER_REPO} -u ${DOCKER_CREDS_USR} --password-stdin
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
                    cat config/deploy/verrazzano-platform-operator.yaml | sed -e "s|IMAGE_NAME|${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_PLATFORM_IMAGE_NAME}:${DOCKER_IMAGE_TAG}|g" > deploy/operator.yaml
                    cat config/crd/bases/install.verrazzano.io_verrazzanos.yaml >> deploy/operator.yaml
                    cat deploy/operator.yaml
                   """
            }
        }

        stage('Update operator.yaml') {
            when {
                allOf {
                    not { buildingTag() }
                    anyOf { branch 'master'; branch 'develop' }
                }
            }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano/platform-operator
                    git config --global credential.helper "!f() { echo username=\\$DOCKER_CREDS_USR; echo password=\\$DOCKER_CREDS_PSW; }; f"
                    git config --global user.name $DOCKER_CREDS_USR
                    git config --global user.email "${DOCKER_EMAIL}"
                    git checkout -b ${env.BRANCH_NAME}
                    git add deploy/operator.yaml
                    git commit -m "[verrazzano] Update verrazzano-platform-operator image version to ${DOCKER_IMAGE_TAG} in operator.yaml"
                    git push origin ${env.BRANCH_NAME}
                   """
            }
            post {
                unsuccessful {
                    script {
                        currentBuild.description = "Git commit of operator.yaml failed"
                    }
                }
            }
        }

        stage('Build') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano
                    make docker-push VERRAZZANO_PLATFORM_OPERATOR_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} VERRAZZANO_APPLICATION_OPERATOR_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
                   """
            }
        }

        stage('gofmt Check') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano/platform-operator
                    make go-fmt
                    cd ${GO_REPO_PATH}/verrazzano/application-operator
                    make go-fmt
                """
            }
        }

        stage('go vet Check') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano/platform-operator
                    make go-vet
                    cd ${GO_REPO_PATH}/verrazzano/application-operator
                    make go-vet
                """
            }
        }

        stage('golint Check') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano/platform-operator
                    make go-lint
                    cd ${GO_REPO_PATH}/verrazzano/application-operator
                    make go-lint
                """
            }
        }

        stage('ineffassign Check') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano/platform-operator
                    make go-ineffassign
                    cd ${GO_REPO_PATH}/verrazzano/application-operator
                    make go-ineffassign
                """
            }
        }

        stage('Third Party License Check') {
            when { not { buildingTag() } }
            steps {
                dir('platform-operator'){
                    echo "In Operator"
                    thirdpartyCheck()
                }
                dir('application-operator'){
                    echo "In OAM Operator"
                    thirdpartyCheck()
                }
            }
        }

        stage('Copyright Compliance Check') {
            when { not { buildingTag() } }
            steps {
                copyrightScan "${WORKSPACE}"
            }
        }

        stage('Unit Tests') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano/platform-operator
                    make -B coverage
                    cp coverage.html ${WORKSPACE}
                    cp coverage.xml ${WORKSPACE}
                    build/scripts/copy-junit-output.sh ${WORKSPACE}
                    cd ${GO_REPO_PATH}/verrazzano/application-operator
                    make -B coverage
                """

                // NEED To See how these files can be merged
                //                    cp coverage.html ${WORKSPACE}
                //                    cp coverage.xml ${WORKSPACE}
                //                    application-operator/build/scripts/copy-junit-output.sh ${WORKSPACE}
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
                      lineCoverageTargets: '85, 85, 85',
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
                    make integ-test DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${DOCKER_PLATFORM_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG}
                    build/scripts/copy-junit-output.sh ${WORKSPACE}
                    cd ${GO_REPO_PATH}/verrazzano/application-operator
                    make integ-test DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${DOCKER_OAM_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG}
                    build/scripts/copy-junit-output.sh ${WORKSPACE}
                """
            }
            post {
                always {
                    archiveArtifacts artifacts: '**/coverage.html,**/logs/*', allowEmptyArchive: true
                    junit testResults: '**/*test-result.xml', allowEmptyResults: true
                }
            }
        }

        stage('Skip acceptance tests if commit message contains skip-at') {
            steps {
                script {
                    // note that SKIP_ACCEPTANCE_TESTS will be false at this point (its default value)
                    // so we are going to run the AT's unless this logic decideds to skip them...

                    // if we are planning to run the AT's (which is the default)
                    if (params.RUN_ACCEPTANCE_TESTS == true) {
                        // check if the user has asked to skip AT using the commit message
                        result = sh (script: "git log -1 | grep 'skip-at'", returnStatus: true)
                        if (result == 0) {
                            // found 'skip-at', so don't run them
                            env.SKIP_ACCEPTANCE_TESTS = "true"
                            echo "Skip acceptance tests based on opt-out in commit message [skip-at]"
                        }
                    }
                    sh """
                        echo "MARK  DEBUGGING"
                        echo "SKIP_ACCEPTANCE_TESTS is $env.SKIP_ACCEPTANCE_TESTS"
                        echo "RUN_ACCEPTANCE_TESTS is $params.RUN_ACCEPTANCE_TESTS"
                    """
                }
            }
        }

        stage('Kick off KinD Merge Acceptance tests') {
            when {
                allOf {
                    not { buildingTag() }
                    anyOf {
                        branch 'master';
                        branch 'develop';
                        expression { return env.SKIP_ACCEPTANCE_TESTS == "false" };
                    }
                }
            }
            environment {
                FULL_IMAGE_NAME = "${DOCKER_PLATFORM_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
            }
            steps {
                build job: "verrazzano-merge-tests/${env.BRANCH_NAME.replace("/", "%2F")}",
                        parameters: [string(name: 'VERRAZZANO_BRANCH', value: env.BRANCH_NAME),
                                     string(name: 'ACCEPTANCE_TESTS_BRANCH', value: params.ACCEPTANCE_TESTS_BRANCH),
                                     string(name: 'VERRAZZANO_OPERATOR_IMAGE', value: FULL_IMAGE_NAME),
                                     string(name: 'TEST_ENV', value: 'kind'),
                                     string(name: 'INSTALL_PROFILE', value: 'dev')],
                        wait: true,
                        propagate: true
            }
        }
    }

    post {
        always {
            deleteDir()
        }
        failure {
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
    }
}
