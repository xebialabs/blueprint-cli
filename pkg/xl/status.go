
package xl

import (
	"fmt"
	"os"
)

var IsQuiet = false
var IsVerbose = false

func Verbose(format string, a ...interface{}) {
	if IsVerbose {
		fmt.Printf(format, a...)
	}
}

func Info(format string, a ...interface{}) {
	if !IsQuiet {
		fmt.Printf(format, a...)
	}
}

func Error(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	os.Exit(1)
}
