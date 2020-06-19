// Copyright (c) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
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

    stages {
        stage('Run acceptance tests on OKE') {
            steps {
                build job: 'verrazzano-oke-acceptance-test-suite/master', parameters: [string(name: 'VERRAZZANO_BRANCH_OR_TAG', value: env.BRANCH_NAME)], wait: true, propagate: true
            }
        }
    }
}
