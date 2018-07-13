pipeline {
  agent {
    docker {
      image 'jamesl22/lit-ci'
    }
  }
  stages {
    stage('Download Deps') {
      steps {
        sh 'make goget'
      }
    }
    stage('Initial Build') {
      steps {
        sh 'make lit'
        sh 'make lit-af'
      }
    }
    stage('Test') {
      steps {
        sh 'make test with-python=true'
      }
    }
    stage('Package') {
      steps {
        sh 'make package'
      }
    }
  }
  post {
    always {
      archiveArtifacts artifacts: 'build/_releasedir/*', fingerprint: false
    }
  }
}
