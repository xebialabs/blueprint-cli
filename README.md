# XL CLI

Project for XL DevOps as Code command line interface.

## Environment

If you are using JetBrains IDE (IntelliJ IDEA Ultimate edition/GoLand), please do this 2 simple steps:

* Install dependencies using gradle (see above)
* In Intellij Idea go to: `Preferences -> Languages & Frameworks -> GO -> GOPATH`
* In Project GOPATH section add click "add" and and select `PROJECT_ROOT/.gogradle/project_gopath`. 
If you cannot see hidden files on MAC, just press `CMD + SHIFT + .`

## Usage

To install dependencies, run:
```
./gradlew vendor
```
If you add a new dependency to the gradle build run:
```
./gradlew vendor lock -Dgogradle.mode=DEV 
```

To build the project, run:
```
./gradlew clean build
```
The xl binary is then created in the folder `xl-cli/build/${GOOS}-${GOARCH}/xl`

## Testing

The Gradle build task is set up to automatically executes the test task. The command to manually run Go tests with Gradle is:
```
./gradlew test
```

To run Go tests using the Go binary, change to the directory containing the test file(s) and run:
```
go test -v
```
From this same directory, you can also see test coverage by running:
```
go test -coverprofile cover.out
go tool cover -html=cover.out
```

## Optimising binaries

There are two ways to optimise the output binaries sizes: 

* You can strip debugging information by passing `-Poptimise`.
  * This will shrink the binary by ~20%, but debugging the binary will be harder.
* You can compress binaries by running upx task `gradle clean build upx`
  * This will shrink the binary by ~50%

In order to run UPX you need to have it installed on your system. The tool is available on brew, yum and apt.

For more information about UPX see: https://upx.github.io

## Release
To release project (git tag), run:
```
./gradlew release -Prelease.versionIncrementer=incrementRule
```
Default rule (without versionIncrementer flag),which increments prerelease number, can be combined with -D.release.prereleasePhase=rc, 
which will change (or start) the prerelease phase (e.g. from alpha to rc). Only one version (tag) possible per commit.

Other incrementer rules are:
* incrementPatch
* incrementMinor
* incrementMajor
* incrementMinorIfNotOnRelease
* incrementPrerelease

Examples:

Starting point is xl-client.8.2.0-alpha.3

./gradlew release

-> xl-client.8.2.0-alpha.4

./gradlew release -Prelease.versionIncrementer=incrementPrerelease

-> xl-client.8.2.0-alpha.5

./gradlew release -Drelease.prereleasePhase=rc

-> xl-client.8.2.0-rc.1

./gradlew release

-> xl-client.8.2.0-rc.2

./gradlew release -Prelease.versionIncrementer=incrementPatch

-> xl-client.8.2.1

./gradlew release -Prelease.versionIncrementer=incrementMinor

-> xl-client.8.3.0

./gradlew release -Prelease.versionIncrementer=incrementMajor

-> xl-client.9.0.0

./gradlew release -Drelease.prereleasePhase=alpha

-> xl-client.9.0.0-alpha.0

Authorization flags:
* SSH: -Prelease.customKeyFile="./keys/secret_key_rsa" -Prelease.customKeyPassword=password
* basic auth: -Prelease.customUsername=username -Prelease.customPassword=password

Jenkins integration flags: ./gradlew release -Prelease.disableChecks -Prelease.pushTagsOnly

Other documentation on: http://axion-release-plugin.readthedocs.io/en/latest/
