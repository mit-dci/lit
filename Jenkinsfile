pipeline {
  agent {
    docker {
      image 'jamesl22/lit-ci'
    }
  }
  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }
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
    stage('Unit Tests') {
      steps {
        sh './scripts/gotests.sh'
      }
    }
    stage('Integration Tests') {
      steps {
        sh 'cd test && env LIT_OUTPUT_SHOW=1 ./runtests.sh'
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
      deleteDir()
    }
  }
}
