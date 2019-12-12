# XL CLI

Project for XL DevOps as Code command line interface. Released binaries of XL CLI can/should be found at https://dist.xebialabs.com/public/xl-cli/.

## Environment

If you are using JetBrains IDE (IntelliJ IDEA Ultimate edition/GoLand), please do this 2 simple steps:

* Install dependencies using gradle (see above)
* In Intellij Idea go to: `Preferences -> Languages & Frameworks -> GO -> GOPATH`
* In Project GOPATH section add click "add" and and select `PROJECT_ROOT/.gogradle/project_gopath`. 
If you cannot see hidden files on MAC, just press `CMD + SHIFT + .`

## Usage

To install dependencies, run:
```
./gradlew goVendor
```
If you add a new dependency to the gradle build run: 
```
./gradlew goVendor goLock -Dgogradle.mode=DEV 
```

To build the project, run:
```
./gradlew goClean goBuild
```
The xl binary is then created in the folder `xl-cli/build/${GOOS}-${GOARCH}/xl`

## Testing

The Gradle build task is set up to automatically executes the test task. The command to manually run Go tests with Gradle is:
```
./gradlew goTest
```

To run Go tests using the Go binary, change to the directory containing the test file(s) and run:
```
go goTest -v
```
From this same directory, you can also see test coverage by running:
```
go goTest -coverprofile cover.out
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
* You can compress binaries by running upx task `gradle goClean goBuild upx`
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

### Bundling license information

In order to comply with some open source licenses, we need to redistribute licenses of open source software that we bundle with our software. This is true for the licenses of most open source license types (like MIT). This holds for our dependencies but ALSO all transitive dependencies. Licenses are made available to our users by running the `xl license` command.

The license data is stored in the `$PROJECT/licenses` folder. The licenses data is populated using a tool called [glice](https://github.com/ribice/glice). The resulting files are checked into version control to cache them. They only need to be updated when our dependencies change, this includes transitive dependencies. We keep them in version control to avoid fetching this information from the internet every build.

When changing/updating dependencies:
* Make sure glice is installed. 
```
go get github.com/ribice/glice
go install github.com/ribice/glice
```

You will need to have your GitHub access token in your `gradle.properties` file, the name of the property is `githubOauthToken`. To get your GitHub access token do the following:

1. Go to [github](https://github.com) and login into your account.
2. Open the settings of your account.
3. Go to `Developers settings` option on the left menu
4. Click on `Personal access tokens`
5. Click on `Generate new token`
6. Add a token description and click `Generate token` (you don't need to add any permissions)
7. Copy your token and put in your `gradle.properties` file. E.g: `githubOauthToken = 78d3d2eff2c9499e33f53870ddda4e9e40420981`

Run `./gradlew updateLicenses` in the root of the project to update license files. Commit changes to version control.  

**NOTE 1**
Please make sure that you use the correct GitHub access token. Without it or with wrong one the `glice` command may *SILENTLY* produce incomplete or empty results.

**NOTE 2** The license data generated from glice needs to be manually verified by you. The tool works on heuristics and not on facts. So everytime you make a change please verify:
* The license of your dependency or transitive dependency is really added
* The license of that specific dependency is detected and of the correct type
  * Go to the library on github and check their license
* The license is not of a type that will prevent us from able to use the library because of for example copyleft rules (like GPL)
* Verify all output of `xl license` is correct after building.

We use [packr](https://github.com/gobuffalo/packr) to embed the license text files into our application. The build will automatically do this for you. Note that a file called `a_cmd-packr.go` is generated. This file should NOT be in version control since its a derived file and could get out of sync. Its excluded from the `gitignore` file. 

Be aware of the limitations of the approach taken:
* Based on heuristics so we need to verify all output
* No unique detection, the license data contains similar or equal license texts
* Missing dependencies. The information is incomplete. Some licenses could not be detected automatically, but also the list of reported libraries by go tools is incomplete (long story) but the best source of info i could find so far


## xl-up

To test the blueprint locally you need `xl cli` on your machine  and also clone this repository:

```$xslt
xl up -b xl-infra -l /PATH/TO/xl-up-blueprint/
```

When you make a PR and also want to run integration test against an existing EKS cluster, GKE or Plain mutlinode K8s cluster then add label ``run-xl-up-pr`` in your github PR, then this pr will run against master in xl-up-blueprint branch. If you are working in story that has changes also in xl-up-blueprint repo in a branch with the same name then also add ``same-branch-on-xl-up-blueprint`` label in your pr

To specify default versions of XLDeploy and XLRelease supported by xl up while building cli use below flags,

```$xslt
./gradlew goClean goBuild  -PXLDVersions=9.0.6 -PXLRVersions=9.0.9
```
