#!groovy
lit('lit') {
currentBuild.result = 'SUCCESS'
  try {
  stage('Test') {

    echo "Building lit"
    sh 'make lit --with-python'
   }
 } catch(err){
   currentBuild.result = "FAILURE"
   throw err
 }
}
