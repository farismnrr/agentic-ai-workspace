// Jenkins version of the Full release, without the dev -> main pull request:
// build `main` as it is, verify it, then publish both images to GHCR as latest and sha-<short>.
//
// Agent needs: git, make, Go (version in go.mod), Node.js 24 + npm, curl and a Docker daemon.
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

    stage('Verify') {
      steps {
        sh 'make verify'
      }
    }

    stage('Binary') {
      steps {
        sh '''
          set -eu
          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make backend-build
          sh scripts/smoke-test.sh bin/agentic-ai-workspace
        '''
      }
    }

    stage('Build images') {
      steps {
        sh '''
          set -eu
          docker build -t "$IMAGE:latest" -t "$IMAGE:sha-$SHORT_SHA" .
          sh scripts/smoke-test-image.sh "$IMAGE:latest"
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
