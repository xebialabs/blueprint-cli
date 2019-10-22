// Exported from:        https://xl-release.xebialabs.com/#/templates/Folder3981784fa264be9b0cb046483b0afa9-Releaseb346c36b186043278aa1a30c3c614561/releasefile
// XL Release version:   9.0.6
// Date created:         Tue Oct 22 16:25:37 CEST 2019

xlr {
  template('Release XL CLI') {
    folder('Developers Love')
    variables {
      stringVariable('EXPLICIT_XLCLI_VERSION') {
        required false
        label 'Explicit XL Cli Version'
        description 'Provide value if you want explicit version i.e. 8.5.0. This will override stage and scope'
      }
      stringVariable('BRANCH_TO_BUILD') {
        label 'Branch to be built and released'
        description 'Name of the branch to build/release'
        value 'master'
      }
      listBoxVariable('RELEASE_STAGE') {
        label 'Release stage'
        description 'Release scope'
        possibleValues 'alpha', 'rc', 'final'
        value 'alpha'
      }
      listBoxVariable('RELEASE_SCOPE') {
        required false
        label 'Release scope'
        possibleValues 'patch', 'minor', 'major'
        value 'patch'
      }
      booleanVariable('CREATE_MAINTENANCE_BRANCH') {
        required false
        label 'Create maintenance branch'
        description 'Create maintenance branch'
      }
      stringVariable('EXPLICIT_COMMIT_ID') {
        required false
        label 'Commit Id for XL CLI'
        description 'Explicit commit ID in case of being needed for XL CLI'
      }
      stringVariable('INTERNAL_XLCLI_VERSION') {
        required false
        showOnReleaseStart false
        label 'XL Cli Version'
      }
      stringVariable('INTERNAL_RELEASE_SCOPE') {
        required false
        showOnReleaseStart false
      }
      stringVariable('INTERNAL_RELEASE_STAGE') {
        required false
        showOnReleaseStart false
      }
      stringVariable('SUPPORTED_XLD_VERSIONS') {
        label 'Supported XLD Versions'
        description 'The XLD versions supported by XL-UP command. Comma separated versions in semver format. E.g: 9.0.5,9.5'
      }
      stringVariable('SUPPORTED_XLR_VERSIONS') {
        label 'Supported XLR Versions'
        description 'The XLR versions supported by XL-UP command. Comma separated versions in semver format. E.g: 9.0.5,9.5'
      }
    }
    scheduledStartDate Date.parse("yyyy-MM-dd'T'HH:mm:ssZ", '2018-10-16T09:00:00+0200')
    phases {
      phase('XL - Cli') {
        color '#0099CC'
        tasks {
          script('Set Variables') {
            script (['''\
if releaseVariables['EXPLICIT_XLCLI_VERSION']:
    releaseVariables['INTERNAL_RELEASE_SCOPE'] = ' '
    releaseVariables['INTERNAL_RELEASE_STAGE'] = ' '
else:
    releaseVariables['INTERNAL_RELEASE_SCOPE'] = releaseVariables['RELEASE_SCOPE']
    releaseVariables['INTERNAL_RELEASE_STAGE'] = releaseVariables['RELEASE_STAGE']
'''])
          }
          custom('Build, release and upload XL - Cli') {
            script {
              type 'jenkins.Build'
              jenkinsServer 'Jenkins NG'
              jobName 'XL Devops As Code/job/XL Cli release/job/Release XL Cli on ${BRANCH_TO_BUILD}'
              jobParameters 'RELEASE_SCOPE=${INTERNAL_RELEASE_SCOPE}\n' +
              'RELEASE_STAGE=${INTERNAL_RELEASE_STAGE}\n' +
              'COMMIT_ID=${EXPLICIT_COMMIT_ID}\n' +
              'RELEASE_EXPLICIT=${EXPLICIT_XLCLI_VERSION}\n' +
              'RELEASE_CREATE_BRANCH=${CREATE_MAINTENANCE_BRANCH}\n' +
              'SUPPORTED_XLD_VERSIONS=${SUPPORTED_XLD_VERSIONS}\n' +
              'SUPPORTED_XLR_VERSIONS=${SUPPORTED_XLR_VERSIONS}'
              jobEnvVarName 'version'
              jobEnvVarValue variable('INTERNAL_XLCLI_VERSION')
            }
          }
          custom('Update Dependencies') {
            script {
              type 'jenkins.Build'
              jenkinsServer 'Jenkins NG'
              jobName 'Update dependencies'
              jobParameters 'branch=${BRANCH_TO_BUILD}\n' +
              'project=groupUpdateAllDependencies\n' +
              'dependency=xlCliVersion\n' +
              'newValue=${INTERNAL_XLCLI_VERSION}'
            }
          }
        }
      }
      phase('Verification and Sync') {
        color '#0099CC'
        tasks {
          parallelGroup('Verify if binary is available') {
            tasks {
              custom('Verify URL for Darwin build') {
                script {
                  type 'webhook.UrlCheck'
                  url 'https://s3.amazonaws.com/xl-cli/bin/${INTERNAL_XLCLI_VERSION}/darwin-amd64/xl'
                }
              }
              custom('Verify URL for Linux build') {
                script {
                  type 'webhook.UrlCheck'
                  url 'https://s3.amazonaws.com/xl-cli/bin/${INTERNAL_XLCLI_VERSION}/linux-amd64/xl'
                }
              }
              custom('Verify URL for Windows build') {
                script {
                  type 'webhook.UrlCheck'
                  url 'https://s3.amazonaws.com/xl-cli/bin/${INTERNAL_XLCLI_VERSION}/windows-amd64/xl.exe'
                }
              }
            }
          }
          sequentialGroup('Sync and verify') {
            precondition 'lowerVersion = releaseVariables[\'INTERNAL_XLCLI_VERSION\'].lower()\n' +
            'result = not ("rc" in lowerVersion or "alpha" in lowerVersion or "alpa" in lowerVersion)'
            tasks {
              custom('Sync XL-CLI to distribution servers') {
                script {
                  type 'remoteScript.Unix'
                  script 'XL_VERSION=${INTERNAL_XLCLI_VERSION} bash xl-rsync.sh xlclient_version.rsync'
                  remotePath '/home/xebialabs/xl-rsync'
                  temporaryDirectoryPath '/home/xebialabs/.tmp'
                  address 'nexus1.xebialabs.cyso.net'
                  username 'xebialabs'
                  password '{aes:v0}3cQdRPGIgz7HID6Mgqvwl1qRkekh0wgjpjMu5h9xBixKwj9/nzs9b6f+ZiMQdFiX'
                }
              }
              custom('Rename files') {
                script {
                  type 'remoteScript.Unix'
                  script 'mkdir /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/windows-amd64\n' +
                  'mkdir /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/darwin-amd64\n' +
                  'mkdir /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/linux-amd64\n' +
                  'mv /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/xl-client-${INTERNAL_XLCLI_VERSION}-windows-amd64.exe /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/windows-amd64/xl.exe\n' +
                  'mv /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/xl-client-${INTERNAL_XLCLI_VERSION}-linux-amd64.bin /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/linux-amd64/xl\n' +
                  'mv /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/xl-client-${INTERNAL_XLCLI_VERSION}-darwin-amd64.bin /var/www/dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/darwin-amd64/xl'
                  remotePath '/home/xldown/.tmp'
                  temporaryDirectoryPath '/home/xldown/.tmp'
                  address 'downloads1.xebialabs.cyso.net'
                  username 'xldown'
                  password '{aes:v0}HwQBb+iEeCYUf1a2aiWijlkqvOmKW/UDsfscuub0VOE='
                }
              }
              parallelGroup('Verify') {
                tasks {
                  custom('Verify URL for Windows build in public') {
                    script {
                      type 'webhook.UrlCheck'
                      url 'https://dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/windows-amd64/xl.exe'
                    }
                  }
                  custom('Verify URL for Linux build in public') {
                    script {
                      type 'webhook.UrlCheck'
                      url 'https://dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/linux-amd64/xl'
                    }
                  }
                  custom('Verify URL for Darwin build in public') {
                    script {
                      type 'webhook.UrlCheck'
                      url 'https://dist.xebialabs.com/public/xl-cli/${INTERNAL_XLCLI_VERSION}/darwin-amd64/xl'
                    }
                  }
                }
              }
            }
          }
          createRelease('Release blueprints') {
            newReleaseTitle 'Blueprints ${INTERNAL_XLCLI_VERSION} release'
            template 'Cloud Love/Blueprints release template'
            folder 'Cloud Love'
            templateVariables {
              stringVariable('BLUEPRINTS_VERSION') {
                label 'Blueprints Version'
                description 'The release version to use, should be x.x.x notation'
                value '${INTERNAL_XLCLI_VERSION}'
              }
              stringVariable('GIT_BRANCH') {
                label 'Git branch'
                description 'Git branch to build'
                value '${BRANCH_TO_BUILD}'
              }
            }
          }
          createRelease('Release XL UP Blueprints') {
            newReleaseTitle 'XL UP Blueprints   release'
            template 'xl-up/XL UP Blueprints release template'
            folder 'xl-up'
            templateVariables {
              stringVariable('BLUEPRINTS_VERSION') {
                label 'Blueprints Version'
                description 'The release version to use, should be x.x.x notation'
                value '${INTERNAL_XLCLI_VERSION}'
              }
              stringVariable('GIT_BRANCH') {
                label 'Git branch'
                description 'Git branch to build'
                value '${BRANCH_TO_BUILD}'
              }
            }
          }
        }
      }
    }

  }
}
