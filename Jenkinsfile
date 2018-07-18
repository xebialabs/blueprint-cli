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
        stage('Build XL Cli') {
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
                sh "./gradlew clean build"
            }
        }
    }
    post {
        success {
            script {
                if(env.BRANCH_NAME == 'master'){
                    hipchatSend color: 'GREEN', credentialId: 'hipchat-token', message: "XL Cli master build <b>SUCCESS</b> - <a href=\"${BUILD_URL}\">click to open</a>", notify: false, room: 'Developer ❤️'
                }
            }
        }
        failure {
            script {
                if(env.BRANCH_NAME == 'master'){
                    hipchatSend color: 'RED', credentialId: 'hipchat-token', message: "XL Cli master build <b>FAILED</b> - <a href=\"${BUILD_URL}\">click to open</a>", notify: true, room: 'Developer ❤️'
                }
            }
        }
    }
}
