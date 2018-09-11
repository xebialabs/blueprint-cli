
package xl

import (
	"bufio"
	"fmt"
	"path/filepath"
	"os"
	"github.com/mattn/go-isatty"
	"gopkg.in/cheggaaa/pb.v1"
)

var IsQuiet = false
var IsVerbose = false
var showingBar = false
var bar *pb.ProgressBar = nil

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
	finishBar()
	fmt.Fprintf(os.Stderr, format, a...)
}

func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	os.Exit(1)
}

func StartProgress(filename string) (error) {
	if !IsQuiet {
		if !IsVerbose && isatty.IsTerminal(os.Stdout.Fd()) {
			nrOfDocs, err := estimateNrOfDocs(filename)
			if err != nil {
				return err
			}
			bar = pb.StartNew(nrOfDocs)
			bar.Prefix(filepath.Base(filename))
			showingBar = true
		} else {
			Info("Applying %s\n", filename)
		}
	}
	return nil
}


func UpdateProgressStartDocument(filename string, doc *Document) {
	if showingBar {
		bar.Increment()
	} else {
		Verbose("... applying document at line %d\n", doc.Line)
	}
}

func UpdateProgressEndDocument() {
	if !showingBar {
		Verbose("... done\n")
	}
}

func EndProgress() {
	if showingBar {
		finishBar()
	} else {
		Verbose("Done\n")
	}
}

func finishBar() {
	if showingBar {
		bar.Set64(bar.Total)
		bar.Finish()
		showingBar = false
	}
}

func estimateNrOfDocs(filename string) (int, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	nrOfDocs := 1
	scanner := bufio.NewScanner(bufio.NewReader(f))
	for scanner.Scan() {
		if scanner.Text() == "---" {
			nrOfDocs++
		}
	}
	return nrOfDocs, nil
}
