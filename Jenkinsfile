// Jenkins version of the Full release, without the dev -> main pull request:
// build `main` as it is, verify it, then publish both images to GHCR as latest and sha-<short>.
//
// Agent needs: git and Docker. Node.js and Go run in containers that mount the workspace at the same
// path, so the Docker daemon must share /var/jenkins_home with Jenkins (the usual dind setup).
// Credential: "ghcr-credentials" (Username with password; the password is a token with write:packages).
pipeline {
  agent any

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
    stage('Checkout main') {
      steps {
        // Explicit URL instead of `checkout scm`, so it also works as an inline "Pipeline script" job.
        git url: 'https://github.com/farismnrr/agentic-ai-workspace.git', branch: 'main'
        script {
          env.SHORT_SHA = sh(script: 'git rev-parse --short=7 HEAD', returnStdout: true).trim()
        }
      }
    }

    // Runs as the Jenkins user so the files it writes stay Jenkins' own.
    stage('Verify frontend') {
      steps {
        sh '''
          docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp -e CI=true \
            -v "$WORKSPACE:$WORKSPACE" -w "$WORKSPACE" "$NODE_IMAGE" make frontend-verify
        '''
      }
    }

    stage('Verify backend and binary') {
      steps {
        sh '''
          docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp -e CI=true \
            -v "$WORKSPACE:$WORKSPACE" -w "$WORKSPACE" "$GO_IMAGE" sh -ec '
              make guardrails backend-verify
              CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make backend-build
              sh scripts/smoke-test.sh bin/agentic-ai-workspace
            '
        '''
      }
    }

    stage('Build images') {
      steps {
        sh '''
          set -eu
          docker build -t "$IMAGE:latest" -t "$IMAGE:sha-$SHORT_SHA" .
          # The smoke test curls 127.0.0.1, so it runs on the daemon's network, next to the published port.
          docker run --rm --network host -v /var/run/docker.sock:/var/run/docker.sock \
            -v "$WORKSPACE:$WORKSPACE" -w "$WORKSPACE" "$DOCKER_CLI_IMAGE" \
            sh -ec 'apk add --no-cache curl >/dev/null; sh scripts/smoke-test-image.sh "$0"' "$IMAGE:latest"
          docker build -f sandbox/Dockerfile -t "$IMAGE-sandbox:latest" -t "$IMAGE-sandbox:sha-$SHORT_SHA" .
          sh scripts/smoke-test-sandbox.sh "$IMAGE-sandbox:latest"
        '''
      }
    }

    stage('Push to GHCR') {
      steps {
        withCredentials([usernamePassword(credentialsId: 'ghcr-credentials',
                                          usernameVariable: 'GHCR_USER',
                                          passwordVariable: 'GHCR_TOKEN')]) {
          sh '''
            set -eu
            echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USER" --password-stdin
            for name in "$IMAGE" "$IMAGE-sandbox"; do
              docker push "$name:latest"
              docker push "$name:sha-$SHORT_SHA"
            done
          '''
        }
      }
    }
  }

  post {
    always {
      sh 'docker logout ghcr.io || true'
    }
  }
}
