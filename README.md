# as-code-cli

Project for XL as-code command line interface.

## Environment

If you are using JetBrains IDE (IntelliJ IDEA Ultimate edition/GoLand), locate this project inside parent directory ~/go/src/github.com/xebialabs,
otherwise, your IDE won't recognize installed dependencies.

## Usage

To install dependencies, run:
```
./gradlew vendor
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
