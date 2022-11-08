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
            def imageName = "wschaefer42/devops-hello-go"
            sh "echo 'build docker image $imageName'"
            sh 'docker build -t $imageName .'
            sh 'docker push $imageName'
            sh 'docker run --name hello-go --rm --network my-net -p 8002:8001 -e REDIS=redis:6379 $imageName'
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
                    sh "CALC_URL=http://localhost:8002 godog"
                }
          }
      }
    }
  }
}