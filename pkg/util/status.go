package util

import (
	"fmt"
	"os"
	"strings"
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

func Indent(step int) string {
	return strings.Repeat(" ", step)
}

func IndentByChunks(step int) string {
	return Indent(4 * step)
}

func Indent1() string {
	return IndentByChunks(1)
}

func Indent2() string {
	return IndentByChunks(2)
}

func Indent3() string {
	return IndentByChunks(3)
}

func IndentFlexible() string {
	if IsVerbose {
		return Indent2()
	}
	return Indent1()
}
