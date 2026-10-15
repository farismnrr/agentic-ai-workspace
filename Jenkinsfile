// Jenkins release pipeline (no dev -> main pull request): checkout -> test -> build -> push -> deploy.
// The deploy hot-swaps the binary in the running container (scripts/deploy-binary.sh) instead of
// rebuilding or replacing it, and puts the previous binary back if the new one does not come up.
//
// Agent needs: git, curl and Docker. Node.js and Go run in containers that mount the workspace at the
// same path, so the Docker daemon must share /var/jenkins_home with Jenkins (the usual dind setup).
// Credential: "ghcr-credentials" (Username with password; the password is a token with write:packages).
pipeline {
  agent any

  parameters {
    string(name: 'BRANCH', defaultValue: 'main', description: 'Branch to build; only main moves the latest tag.')
  }

  options {
    disableConcurrentBuilds()
    timeout(time: 45, unit: 'MINUTES')
    timestamps()
  }

  environment {
    CI = 'true'
    IMAGE = 'ghcr.io/farismnrr/agentic-ai-workspace'
    NODE_IMAGE = 'node:24-bookworm'
    GO_IMAGE = 'golang:1.27.2-bookworm'
    DOCKER_CLI_IMAGE = 'docker:29-cli'
  }

  stages {
    stage('Checkout') {
      steps {
        // Explicit URL instead of `checkout scm`, so it also works as an inline "Pipeline script" job.
        git url: 'https://github.com/farismnrr/agentic-ai-workspace.git', branch: params.BRANCH
        script {
          env.SHORT_SHA = sh(script: 'git rev-parse --short=7 HEAD', returnStdout: true).trim()
          // Commit plus build number, so every run is a distinct, traceable version.
          env.VERSION = "${env.SHORT_SHA}-b${env.BUILD_NUMBER}"
        }
        echo "Building ${params.BRANCH} at ${env.SHORT_SHA} as version ${env.VERSION}"
      }
    }

    // Any failing check stops the pipeline here, before anything is built, pushed or deployed.
    // Containers run as the Jenkins user so the files they write stay Jenkins' own.
    stage('Test') {
      steps {
        sh '''
          docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp -e CI=true \
            -v "$WORKSPACE:$WORKSPACE" -w "$WORKSPACE" "$NODE_IMAGE" make frontend-verify
          docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp -e CI=true \
            -v "$WORKSPACE:$WORKSPACE" -w "$WORKSPACE" "$GO_IMAGE" make guardrails backend-verify
        '''
      }
    }

    stage('Build binary') {
      steps {
        sh '''
          docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp -e CI=true -e VERSION \
            -v "$WORKSPACE:$WORKSPACE" -w "$WORKSPACE" "$GO_IMAGE" sh -ec '
              CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make backend-build
              sh scripts/smoke-test.sh bin/agentic-ai-workspace
            '
        '''
      }
    }

    stage('Build image') {
      steps {
        sh '''
          set -eu
          docker build --build-arg VERSION="$VERSION" -t "$IMAGE:sha-$SHORT_SHA" .
          # The smoke test curls 127.0.0.1, so it runs on the daemon's network, next to the published port.
          docker run --rm --network host -v /var/run/docker.sock:/var/run/docker.sock \
            -v "$WORKSPACE:$WORKSPACE" -w "$WORKSPACE" "$DOCKER_CLI_IMAGE" \
            sh -ec 'apk add --no-cache curl >/dev/null; sh scripts/smoke-test-image.sh "$0"' "$IMAGE:sha-$SHORT_SHA"
          docker build -f sandbox/Dockerfile -t "$IMAGE-sandbox:sha-$SHORT_SHA" .
          sh scripts/smoke-test-sandbox.sh "$IMAGE-sandbox:sha-$SHORT_SHA"
          docker images "$IMAGE"
        '''
      }
    }

    stage('Push') {
      steps {
        withCredentials([usernamePassword(credentialsId: 'ghcr-credentials',
                                          usernameVariable: 'GHCR_USER',
                                          passwordVariable: 'GHCR_TOKEN')]) {
          sh '''
            set -eu
            echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USER" --password-stdin
            for name in "$IMAGE" "$IMAGE-sandbox"; do
              docker push "$name:sha-$SHORT_SHA"
              if [ "$BRANCH" = main ]; then
                docker tag "$name:sha-$SHORT_SHA" "$name:latest"
                docker push "$name:latest"
              fi
            done
          '''
        }
      }
    }

    stage('Deploy') {
      steps {
        sh 'sh scripts/deploy-binary.sh bin/agentic-ai-workspace "$VERSION" "$IMAGE:sha-$SHORT_SHA"'
      }
    }
  }

  post {
    always {
      sh 'docker logout ghcr.io || true'
    }
  }
}
