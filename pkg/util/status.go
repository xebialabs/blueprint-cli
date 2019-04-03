package util

import (
    "fmt"
    "os"
    "sort"
    "strings"
)

var IsQuiet = false
var IsVerbose = false

const (
    TableAlignLeft       string = "left"
    TableAlignRight      string = "right"
)

func DataMapTable(dataMap *map[string]interface{}, align string, keyWidth int, valWidth int, leftSpacer string) string {
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("%s %s %s\n", leftSpacer, strings.Repeat("_", keyWidth), strings.Repeat("_", valWidth)))
    if align == TableAlignLeft {
        sb.WriteString(fmt.Sprintf("%s|%-*s|%-*s|\n", leftSpacer, keyWidth, "KEY", valWidth, "VALUE"))
    } else if align == TableAlignRight {
        sb.WriteString(fmt.Sprintf("%s|%*s|%*s|\n", leftSpacer, keyWidth, "KEY", valWidth, "VALUE"))
    }
    sb.WriteString(fmt.Sprintf("%s %s %s\n", leftSpacer, strings.Repeat("-", keyWidth), strings.Repeat("-", valWidth)))
    keys := ExtractStringKeysFromMap(*dataMap)
    sort.Strings(keys)
    for _, k := range keys {
        // truncate strings if needed
        key := k
        if len(key) > keyWidth {
            key = string(k[:keyWidth-2]) + ".."
        }
        val := strings.Replace(fmt.Sprintf("%v", (*dataMap)[k]), "\n", "\\n", -1)
        if len(val) > valWidth {
            val = string(val[:valWidth-2]) + ".."
        }

        // do alignment
        if align == TableAlignLeft {
            sb.WriteString(fmt.Sprintf("%s|%-*s|%-*s|\n", leftSpacer, keyWidth, key, valWidth, val))
        } else if align == TableAlignRight {
            sb.WriteString(fmt.Sprintf("%s|%*s|%*s|\n", leftSpacer, keyWidth, key, valWidth, val))
        }
    }
    sb.WriteString(fmt.Sprintf("%s %s %s\n", leftSpacer, strings.Repeat("-", keyWidth), strings.Repeat("-", valWidth)))
    return sb.String()
}

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
