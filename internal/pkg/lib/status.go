
package lib

import (
	"fmt"
	"os"
)

var IsVerbose = false

func Verbose(format string, a ...interface{}) {
	if IsVerbose {
		fmt.Printf(format, a...)
	}
}

func Info(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

func Error(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func Fatal(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

