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
The xl binary is then created in the folder `xl-cli/build/output`
