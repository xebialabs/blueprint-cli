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
}
