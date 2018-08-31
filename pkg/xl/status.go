
package xl

import (
	"fmt"
	"os"
	"gopkg.in/cheggaaa/pb.v1"
	"github.com/mattn/go-isatty"
	"bufio"
	"path/filepath"
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
	fmt.Fprintf(os.Stderr, format, a...)
}

func Fatal(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func StartProgress(total int) {
	if !IsQuiet {
		if !IsVerbose && isatty.IsTerminal(os.Stdout.Fd()) {
			bar = pb.StartNew(total)
			showingBar = true
		}
	}
}

func UpdateProgressStartFile(filename string) {
	if !IsQuiet {
		if showingBar {
			bar.Prefix(filepath.Base(filename))
		} else {
			Info("Processing file %s\n", filename)
		}
	}
}

func UpdateProgressStartDocument(filename string, doc *Document) {
	if !IsQuiet {
		if showingBar {
			bar.Increment()
		} else {
			Info("Processing document at line %d\n", doc.Line)
		}
	}
}

func UpdateProgressEndFile() {
	if !IsQuiet {
		if showingBar {
			bar.Prefix("")
		}
	}
}

func EndProgress() {
	if !IsQuiet {
		if showingBar {
			bar.FinishPrint("Done")
		} else {
			Info("Done\n")
		}
	}
}

func CountTotalNrOfDocs(filenames []string) (int, error){
	var totalNrOfDocuments = 0

	for _, filename := range filenames {
		nrOfDocuments, err := EstimateNrOfDocs(filename)
		if err != nil {
			return 0, err
		}
		totalNrOfDocuments += nrOfDocuments
	}

	return totalNrOfDocuments, nil
}

func EstimateNrOfDocs(filename string) (int, error) {
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
