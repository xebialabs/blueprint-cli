#!groovy
@Library('jenkins-pipeline-libs@master')
import com.xebialabs.pipeline.utils.Branches

pipeline {
    agent none

    options {
        buildDiscarder(logRotator(numToKeepStr: '20', artifactDaysToKeepStr: '7', artifactNumToKeepStr: '5'))
        timeout(time: 1, unit: 'HOURS')
        timestamps()
        ansiColor('xterm')
    }

    environment {
        REPOSITORY_NAME = 'xl-cli'
        DIST_SERVER_CRED = credentials('distserver')
        ON_PREM_CERT = "${env.ON_PREM_CERT}"
        ON_PREM_KEY = "${env.ON_PREM_KEY}"
        ON_PREM_K8S_API_URL = "${env.ON_PREM_K8S_API_URL}"
        NSF_SERVER_HOST = "${env.NSF_SERVER_HOST}"
        XL_UP_GCP_PROJECT_ID = "${env.XL_UP_GCP_PROJECT_ID}"
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
                sh "./gradlew clean build sonarqube -Dsonar.branch.name=${getBranch()} --info -x updateLicenses"
                stash name: "xl-up", includes: "build/linux-amd64/xl"
                script {
                  if (fileExists('build/version.dump') == true) {
                    currentVersion = readFile 'build/version.dump'

                    env.version = currentVersion
                  }
                }
                archiveArtifacts artifacts: 'build/*/xl', fingerprint: true
                archiveArtifacts artifacts: 'build/*/xl.exe', fingerprint: true
                archiveArtifacts artifacts: '.gogradle/reports/test/*', fingerprint: true
            }
        }


        stage('Run XL UP Branch Linux') {


            parallel {
                stage('e2e tests on AWS EKS') {
                    agent {
                        label "xld||xlr||xli"
                    }
                    when {
                        expression {
                            !Branches.onMasterOrMaintenanceBranch(env.BRANCH_NAME) &&
                                githubLabelsPresent(this, ['run-xl-up-pr'])
                        }
                    }

                    steps {
                        script {
                            try {
                                sh "mkdir -p temp"
                                dir('temp') {
                                    if (githubLabelsPresent(this, ['same-branch-on-xl-up-blueprint'])){
                                        sh "git clone -b ${CHANGE_BRANCH} git@github.com:xebialabs/xl-up-blueprint.git || true"
                                    } else {
                                        sh "git clone git@github.com:xebialabs/xl-up-blueprint.git || true"
                                    }
                                }
                                unstash name: 'xl-up'
                                awsConfigure = readFile "/var/lib/jenkins/.aws/credentials"
                                awsAccessKeyIdLine = awsConfigure.split("\n")[1]
                                awsSecretKeyIdLine = awsConfigure.split("\n")[2]
                                awsAccessKeyId = awsAccessKeyIdLine.split(" ")[2]
                                awsSecretKeyId = awsSecretKeyIdLine.split(" ")[2]
                                sh "curl https://dist.xebialabs.com/customer/licenses/download/v3/deployit-license.lic -u ${DIST_SERVER_CRED} -o temp/xl-up-blueprint/deployit-license.lic"
                                sh "curl https://dist.xebialabs.com/customer/licenses/download/v3/xl-release-license.lic -u ${DIST_SERVER_CRED} -o temp/xl-up-blueprint/xl-release.lic"
                                eksEndpoint = sh (script: 'aws eks describe-cluster --region eu-west-1 --name xl-up-master --query \'cluster.endpoint\' --output text', returnStdout: true).trim()
                                efsFileId = sh (script: 'aws efs describe-file-systems --region eu-west-1 --query \'FileSystems[0].FileSystemId\' --output text', returnStdout: true).trim()
                                runXlUpOnEks(awsAccessKeyId, awsSecretKeyId, eksEndpoint, efsFileId)
                                sh "rm -rf temp"
                            } catch (err) {
                                sh "rm -rf temp"
                                throw err
                            }
                        }

                    }

                }

                stage('e2e tests on GCP GKE') {
                    agent {
                        label "xld||xlr||xli"
                    }
                    when {
                        expression {
                            !Branches.onMasterOrMaintenanceBranch(env.BRANCH_NAME) &&
                                githubLabelsPresent(this, ['run-xl-up-pr'])
                        }
                    }

                    steps {
                        script {
                            try {
                                sh "mkdir -p temp"
                                dir('temp') {
                                    if (githubLabelsPresent(this, ['same-branch-on-xl-up-blueprint'])){
                                        sh "git clone -b ${CHANGE_BRANCH} git@github.com:xebialabs/xl-up-blueprint.git || true"
                                    } else {
                                        sh "git clone git@github.com:xebialabs/xl-up-blueprint.git || true"
                                    }
                                }
                                unstash name: 'xl-up'
                                sh "curl https://dist.xebialabs.com/customer/licenses/download/v3/deployit-license.lic -u ${DIST_SERVER_CRED} -o temp/xl-up-blueprint/deployit-license.lic"
                                sh "curl https://dist.xebialabs.com/customer/licenses/download/v3/xl-release-license.lic -u ${DIST_SERVER_CRED} -o temp/xl-up-blueprint/xl-release.lic"
                                runXlUpOnGke()
                                sh "rm -rf temp"
                            } catch (err) {
                                sh "rm -rf temp"
                                throw err
                            }
                        }

                    }

                }
                stage('e2e tests on On-Prem') {
                    agent {
                        label "xld||xlr||xli"
                    }
                    when {
                        expression {
                            !Branches.onMasterOrMaintenanceBranch(env.BRANCH_NAME) &&
                                githubLabelsPresent(this, ['run-xl-up-pr'])
                        }
                    }

                    steps {
                        script {
                            try {
                                sh "mkdir -p temp"
                                dir('temp') {
                                    if (githubLabelsPresent(this, ['same-branch-on-xl-up-blueprint'])){
                                        sh "git clone -b ${CHANGE_BRANCH} git@github.com:xebialabs/xl-up-blueprint.git || true"
                                    } else {
                                        sh "git clone git@github.com:xebialabs/xl-up-blueprint.git || true"
                                    }
                                }
                                unstash name: 'xl-up'
                                sh "curl https://dist.xebialabs.com/customer/licenses/download/v3/deployit-license.lic -u ${DIST_SERVER_CRED} -o temp/xl-up-blueprint/deployit-license.lic"
                                sh "curl https://dist.xebialabs.com/customer/licenses/download/v3/xl-release-license.lic -u ${DIST_SERVER_CRED} -o temp/xl-up-blueprint/xl-release.lic"
                                nfsSharePath = "xebialabs-k8s"
                                runXlUpOnPrem(nfsSharePath)
                                sh "rm -rf temp"
                            } catch (err) {
                                sh "rm -rf temp"
                                throw err
                            }
                        }

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

def runXlUpOnEks(String awsAccessKeyId, String awsSecretKeyId, String eksEndpoint, String efsFileId) {
    sh "sed -ie 's@https://aws-eks.com:6443@${eksEndpoint}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@SOMEKEY@${awsAccessKeyId}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@SOMEMOREKEY@${awsSecretKeyId}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@test1234561@${efsFileId}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@test-eks-master@xl-up-master@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XldLic: ./deployit-license.lic@XldLic: temp/xl-up-blueprint/deployit-license.lic@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XlrLic: ./xl-release.lic@XlrLic: temp/xl-up-blueprint/xl-release.lic@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XlKeyStore: ./integration-tests/files/keystore.jceks@XlKeyStore: temp/xl-up-blueprint/integration-tests/files/keystore.jceks@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml"
    sh "./build/linux-amd64/xl up -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --undeploy --skip-prompts"
    sh "./build/linux-amd64/xl up -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --seed-version 9.5.0 --skip-prompts"
    sh "./build/linux-amd64/xl up -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/eks-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --undeploy --skip-prompts"

}


def runXlUpOnPrem(String nfsSharePath) {
    sh """ if [[ ! -f "temp/xl-up-blueprint/k8sClientCert-onprem.crt" ]]; then
        echo ${ON_PREM_CERT} >> temp/xl-up-blueprint/k8sClientCert-onprem-tmp.crt
        tr ' ' '\\n' < temp/xl-up-blueprint/k8sClientCert-onprem-tmp.crt > temp/xl-up-blueprint/k8sClientCert-onprem-tmp2.crt
        tr '%' ' ' < temp/xl-up-blueprint/k8sClientCert-onprem-tmp2.crt > temp/xl-up-blueprint/k8sClientCert-onprem.crt
        rm -f temp/xl-up-blueprint/k8sClientCert-onprem-tmp.crt | rm -f temp/xl-up-blueprint/k8sClientCert-onprem-tmp2.crt
    fi"""

    sh """ if [[ ! -f "temp/xl-up-blueprint/k8sClientCert-onprem.key" ]]; then
        echo ${ON_PREM_KEY} >> temp/xl-up-blueprint/k8sClientCert-onprem-tmp.key
        tr ' ' '\\n' < temp/xl-up-blueprint/k8sClientCert-onprem-tmp.key > temp/xl-up-blueprint/k8sClientCert-onprem-tmp2.key
        tr '%' ' ' < temp/xl-up-blueprint/k8sClientCert-onprem-tmp2.key > temp/xl-up-blueprint/k8sClientCert-onprem.key
        rm -f temp/xl-up-blueprint/k8sClientCert-onprem-tmp.key | rm -f temp/xl-up-blueprint/k8sClientCert-onprem-tmp2.key
    fi"""

    sh "sed -ie 's@https://k8s.com:6443@${ON_PREM_K8S_API_URL}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@K8sClientCertFile: ../xl-up/__test__/files/test-file@K8sClientCertFile: temp/xl-up-blueprint/k8sClientCert-onprem.crt@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@K8sClientKeyFile: ../xl-up/__test__/files/test-file@K8sClientKeyFile: temp/xl-up-blueprint/k8sClientCert-onprem.key@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@nfs-test.com@${NSF_SERVER_HOST}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@/xebialabs@/${nfsSharePath}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XldLic: ./deployit-license.lic@XldLic: temp/xl-up-blueprint/deployit-license.lic@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XlrLic: ./xl-release.lic@XlrLic: temp/xl-up-blueprint/xl-release.lic@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XlKeyStore: ./integration-tests/files/keystore.jceks@XlKeyStore: temp/xl-up-blueprint/integration-tests/files/keystore.jceks@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml"
    sh "./build/linux-amd64/xl up -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --undeploy --skip-prompts"
    sh "./build/linux-amd64/xl up -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --seed-version 9.5.0 --skip-prompts"
    sh "./build/linux-amd64/xl up -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/on-prem-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --undeploy --skip-prompts"
}

def runXlUpOnGke() {
    GKE_ACCOUNT_EMAIL = sh(script: 'cat /var/lib/jenkins/.gcloud/account.json | python -c \'import json, sys; obj = json.load(sys.stdin); print obj["client_email"];\'', returnStdout: true).trim()

    sh "gcloud auth activate-service-account ${GKE_ACCOUNT_EMAIL} --key-file=/var/lib/jenkins/.gcloud/account.json"
    sh "gcloud container clusters get-credentials  gke-xl-up-cluster --zone europe-west3-b --project ${XL_UP_GCP_PROJECT_ID}"

    GKE_ENDPOINT = sh(script: 'kubectl config view --minify -o jsonpath=\'{.clusters[0].cluster.server}\'', returnStdout: true).trim()
    SECRET_NAME = sh(script: "kubectl get secrets -o custom-columns=:metadata.name -n kube-system | grep xebialabs-admin", returnStdout: true).trim()
    GKE_TOKEN = sh(script: "kubectl get secrets --field-selector metadata.name=${SECRET_NAME} -n kube-system -o=jsonpath='{.items[].data.token}' | base64 -d", returnStdout: true).trim()
    NFS_PATH = sh(script: "gcloud filestore instances list --project ${XL_UP_GCP_PROJECT_ID} --format='csv(fileShares.name,networks.ipAddresses[0])' | sed -n 2p | tr ',' '\n' | sed -n 1p", returnStdout: true).trim()
    NFS_HOST = sh(script: "gcloud filestore instances list --project ${XL_UP_GCP_PROJECT_ID} --format='csv(fileShares.name,networks.ipAddresses[0])' | sed -n 2p | tr ',' '\n' | sed -n 2p", returnStdout: true).trim()

    sh "sed -ie 's@{{GKE_ENDPOINT}}@${GKE_ENDPOINT}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@{{K8S_TOKEN}}@${GKE_TOKEN}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@{{NFS_HOST}}@${NFS_HOST}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@{{NFS_PATH}}@/${NFS_PATH}@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XldLic: ./deployit-license.lic@XldLic: temp/xl-up-blueprint/deployit-license.lic@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XlrLic: ./xl-release.lic@XlrLic: temp/xl-up-blueprint/xl-release.lic@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml"
    sh "sed -ie 's@XlKeyStore: ./integration-tests/files/keystore.jceks@XlKeyStore: temp/xl-up-blueprint/integration-tests/files/keystore.jceks@g' temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml"

    sh "./build/linux-amd64/xl up -d -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --undeploy --skip-prompts"
    sh "./build/linux-amd64/xl up -d -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --seed-version 9.5.0 --skip-prompts"
    sh "./build/linux-amd64/xl up -d -a temp/xl-up-blueprint/integration-tests/test-cases/jenkins/gke-xld-xlr-mon-full.yaml -b xl-infra -l temp/xl-up-blueprint/ --undeploy --skip-prompts"
}
