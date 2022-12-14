def imageName = 'wschaefer42/devops-hello-go'

pipeline {
  agent any

  tools { go '1.19' }

  stages {
    stage('Build') {
        steps {
            sh 'go build .'
        }
    }
    stage('Test') {
        steps {
            sh 'go test .'
        }
    }
    stage("Docker") {
        steps {
            catchError {
                sh "docker rm -f hello-go"
            }
            sh "echo 'build docker image $imageName'"
            sh "echo '$PATH'"
            sh "docker build -t ${imageName} ."
            sh "docker push ${imageName}"
            sh "docker run --name hello-go --rm -d --network my-net -p 8002:8001 -e REDIS=redis:6379 ${imageName}"
        }
    }
    stage("Acceptance Tests") {
      parallel {
          stage("Acceptance test script") {
                steps {
                    sleep 60
                    sh "chmod +x acceptance_test.sh && ./acceptance_test.sh"
                }
          }
          stage("Acceptance test cucumber") {
                steps {
                    sh "go install github.com/cucumber/godog/cmd/godog@latest"
                    sh "CALC_URL=http://localhost:8002 `go env GOPATH`/bin/godog run"
                }
          }
      }
    }
  }
}