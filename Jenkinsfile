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
    stage('Unit Tests') {
      steps {
        sh './gotests.sh'
      }
    }
    stage('Integration Tests') {
      steps {
        try {
          sh 'python3 test/test_basic.py -c reg --dumplogs'
          sh 'python3 test/test_break.py -c reg --dumplogs'
        } catch (Exception e) {
          currentBuild.result = 'UNSTABLE'
        }
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
