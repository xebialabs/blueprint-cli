Test
# XL Blueprint CLI

Project for XL Blueprint command line interface.

## Environment

If you are using JetBrains IDE (IntelliJ IDEA Ultimate edition/GoLand), please do this 2 simple steps:

* Install dependencies using gradle (see above)
* In Intellij Idea go to: `Preferences -> Languages & Frameworks -> GO -> GOPATH`
* In Project GOPATH section add click "add" and and select `PROJECT_ROOT/.gogradle/project_gopath`. 
If you cannot see hidden files on MAC, just press `CMD + SHIFT + .`

## Usage

To install dependencies, run:
```
go get name.of/dependency@version
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

When running tests locally using Git blueprint repo fro xl-up, if you hit rate limit errors, set an env variable called `XL_UP_GITHUB_TOKEN` with your Github access token to increase the limit.

## Debugging

It is possible to debug xl-cli with the help of, for example, Intellij Idea.

To do that you need to perform few simple steps:
1) Install **dlv** by following this instructions: https://github.com/go-delve/delve/tree/master/Documentation/installation
2) Compile xl-cli binaries with debug information. For that use -Pdebug flag when building it.<br>Example: `gradle build -Pdebug
3) Run xl-cli in debug mode via **dlv** (assuming that you are currently in the root of xl-cli project):<br>
`dlv --listen=:2345 --headless=true --api-version=2 exec build/darwin-amd64/xl -- apply -v -f provision.yaml`
4) Connect using Intellij Idea. In debug configurations click "+" button and choose "Go Remote". Default parameters must be enough to connect.<br>
After saving it just click small "bug" button to start debug session.

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

Authorization flags:
* SSH: -Prelease.customKeyFile="./keys/secret_key_rsa" -Prelease.customKeyPassword=password
* basic auth: -Prelease.customUsername=username -Prelease.customPassword=password

Jenkins integration flags: ./gradlew release -Prelease.disableChecks -Prelease.pushTagsOnly

Other documentation on: http://axion-release-plugin.readthedocs.io/en/latest/
