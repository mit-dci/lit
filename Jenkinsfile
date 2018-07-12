pipeline {
  agent {
      docker { image 'jamesl22/lit-ci' }
  }
  stages {
    stage('Test lit') {
      steps {
        sh 'rm -rf lit ../lit'
        sh 'git clone https://github.com/mit-dci/lit'
        sh 'mv lit ../'
        sh 'cd ..'
        sh 'make lit lit-af'
        sh 'make test with-python=true'
      }
    }
  }
}
