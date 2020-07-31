// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
def HEAD_COMMIT
pipeline {
    options {
        skipDefaultCheckout true
        disableConcurrentBuilds()
    }

    agent {
        docker {
            image "${RUNNER_DOCKER_IMAGE}"
            args "${RUNNER_DOCKER_ARGS}"
            registryUrl "${RUNNER_DOCKER_REGISTRY_URL}"
            registryCredentialsId 'ocir-pull-and-push-account'
        }
    }

    environment {
        GITHUB_API_TOKEN = credentials('github-api-token-release-assets')
    }

    stages {
        stage('Run acceptance tests on OKE') {
            steps {
                script {
                    checkout scm
                    echo "Last 5 commits are:"
                    sh "git log -n 5"
                    HEAD_COMMIT = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
                    echo "HEAD COMMIT is ${HEAD_COMMIT} and branch is ${env.BRANCH_NAME}"
                    if (env.BRANCH_NAME == "master") {
                        // For master branch, trigger the "pristine" master OKE tests
                        build job: 'verrazzano-oke-acceptance-test-master/master', parameters: [string(name: 'VERRAZZANO_BRANCH_OR_TAG', value: HEAD_COMMIT)], wait: true, propagate: true
                    } else {
                        build job: 'verrazzano-oke-acceptance-test-suite/master', parameters: [string(name: 'VERRAZZANO_BRANCH_OR_TAG', value: HEAD_COMMIT)], wait: true, propagate: true
                    }
                }
            }
        }
    }
}
