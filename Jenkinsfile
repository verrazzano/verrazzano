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
        stage('Run acceptance tests') {
            steps {
                build job: 'verrazzano-acceptance-test-suite/master', parameters: [string(name: 'VERRAZZANO_BRANCH_NAME', value: env.BRANCH_NAME)], wait: true, propagate: true
            }
        }
    }
}
