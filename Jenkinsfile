#!groovy

pipeline {
    agent none

    options {
        buildDiscarder(logRotator(numToKeepStr: '20', artifactDaysToKeepStr: '7', artifactNumToKeepStr: '5'))
        timeout(time: 1, unit: 'HOURS')
        timestamps()
        ansiColor('xterm')
    }

    stages {
        stage('Build XL CLI on Linux') {
            agent {
                node {
                    label 'xld||xlr'
                }
            }

            tools {
                jdk 'JDK 8u171'
            }

            steps {
                checkout scm
                sh "./gradlew goClean goBuild sonarqube -Dsonar.branch.name=${getBranch()} -PincludeXlUp --info -x updateLicenses"
                script {
                  if (fileExists('build/version.dump') == true) {
                    currentVersion = readFile 'build/version.dump'

                    env.version = currentVersion
                  }
                }
            }
        }

    }
    post {
        success {
            script {
                if(env.BRANCH_NAME == 'master'){
                    slackSend color: "good", tokenCredentialId: "slack-token", message: "XL Cli master build *SUCCESS* - <${env.BUILD_URL}|click to open>", channel: 'team-developer-love'
                }
            }
        }
        failure {
            script {
                if(env.BRANCH_NAME == 'master'){
                    slackSend color: "danger", tokenCredentialId: "slack-token", message: "XL Cli master build *FAILED* - <${env.BUILD_URL}|click to open>", channel: 'team-developer-love'
                }
            }
        }
    }
}

def getBranch() {
    // on simple Jenkins pipeline job the BRANCH_NAME is not filled in, and we run it only on master
    return env.BRANCH_NAME ?: 'master'
}
