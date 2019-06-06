package util

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
)

var IsQuiet = false
var IsVerbose = false
var SummaryTableHeaders = [...]string {"LABEL", "VALUE"}

const (
	TableAlignLeft  string = "left"
	TableAlignRight string = "right"
)

func DataMapTable(dataMap *map[string]interface{}, align string, keyWidth int, valWidth int, leftSpacer string, padding int) string {
    var sb strings.Builder

    // prepare formats
    border := fmt.Sprintf("%s %s %s\n",
        leftSpacer,
        strings.Repeat("-", keyWidth + (padding * 2)),
        strings.Repeat("-", valWidth + (padding * 2)),
    )
    var rowFormat string
    if align == TableAlignLeft {
        rowFormat = "%s|%s%-*s%s|%s%-*s%s|\n"
    } else if align == TableAlignRight {
        rowFormat = "%s|%s%*s%s|%s%*s%s|\n"
    }

    // output headers
    sb.WriteString(border)
    sb.WriteString(fmt.Sprintf(rowFormat,
        leftSpacer,
        strings.Repeat(" ", padding),
        keyWidth, SummaryTableHeaders[0],
        strings.Repeat(" ", padding),
        strings.Repeat(" ", padding),
        valWidth, SummaryTableHeaders[1],
        strings.Repeat(" ", padding),
    ))

    // output rows
    sb.WriteString(border)
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
        sb.WriteString(fmt.Sprintf(rowFormat,
            leftSpacer,
            strings.Repeat(" ", padding),
            keyWidth, key,
            strings.Repeat(" ", padding),
            strings.Repeat(" ", padding),
            valWidth, val,
            strings.Repeat(" ", padding),
        ))
    }

    sb.WriteString(border)
    return sb.String()
}

func Verbose(format string, a ...interface{}) {
	if IsVerbose {
		fmt.Printf(format, a...)
	}
}

func Info(format string, a ...interface{}) {
	if IsVerbose || !IsQuiet {
		fmt.Printf(format, a...)
	}
}

func Print(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

func Error(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	os.Exit(1)
}

func Trace(format string, a ...interface{}) {
	if IsVerbose {
		pc := make([]uintptr, 10) // at least 1 entry needed
		runtime.Callers(2, pc)
		f := runtime.FuncForPC(pc[0])
		file, line := f.FileLine(pc[0])
		fmt.Printf("Function %s in file %s:%d\n", f.Name(), file, line)
		fmt.Printf(format, a...)
	}
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
