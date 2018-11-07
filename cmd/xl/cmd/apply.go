package cmd

import (
	"fmt"
	"github.com/deckarep/golang-set"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type FileWithDocuments struct {
	imports   []string
	documents []*xl.Document
	fileName  string
}

var applyFilenames []string
var applyValues map[string]string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes",
	Long:  `Apply configuration changes`,
	Run: func(cmd *cobra.Command, args []string) {
		DoApply(applyFilenames)
	},
}

func printCiIds(kind string, ids *[]string) {
	if ids != nil && len(*ids) > 0 {
		xl.Verbose("...... ---------------\n")
		xl.Verbose(fmt.Sprintf("...... %s CIs:\n", kind))
		xl.Verbose("...... ---------------\n")
		for idx, id := range *ids {
			xl.Verbose(fmt.Sprintf("...... %d. %s\n", idx+1, id))
		}
		xl.Verbose("...... ---------------\n")
	}
}

func printChangedCis(changedCis *xl.ChangedCis) {
	if changedCis != nil {
		printCiIds("Created", changedCis.Created)
		printCiIds("Updated", changedCis.Updated)
	}
}

func printTaskInfo(task *xl.TaskInfo) {
	if task != nil {
		xl.Verbose("...... ---------------\n")
		xl.Verbose(fmt.Sprintf("...... Task [%s] is started:\n", task.Id))
		xl.Verbose(fmt.Sprintf("...... %s.\n", task.Description))
		xl.Verbose("...... ---------------\n")
	}
}

func printChanges(changes *xl.Changes) {
	if changes != nil {
		printChangedCis(changes.Cis)
		printTaskInfo(changes.Task)
	}
}

func checkForEmptyImport(importedFile string) {
	if len(strings.TrimSpace(importedFile)) == 0 {
		xl.Fatal("The 'imports' field contains empty elements.\n")
	}
}

func extractImports(baseDir string, doc *xl.Document) []string {
	if doc.Metadata != nil && doc.Metadata["imports"] != nil {
		if imports, ok := doc.Metadata["imports"].([]interface{}); !ok {
			xl.Fatal("The 'imports' field has wrong format. Must be a list of strings.\n")
		} else {
			delete(doc.Metadata, "imports")
			importedFiles, _ := funk.Map(imports, func(i interface{}) string {
				importedFile, _ := i.(string)
				checkForEmptyImport(importedFile)
				err := xl.ValidateFilePath(importedFile, "imports")
				if err != nil {
					xl.Fatal(err.Error())
				}
				return filepath.Join(baseDir, filepath.FromSlash(importedFile))
			}).([]string)
			return importedFiles
		}
	}
	return make([]string, 0)
}

func readDocumentsFromFile(fileName string) FileWithDocuments {
	reader, err := os.Open(fileName)
	if err != nil {
		xl.Fatal("Error while opening XL YAML file %s: %s\n", fileName, err)
	}
	imports := make([]string, 0)
	documents := make([]*xl.Document, 0)
	docReader := xl.NewDocumentReader(reader)
	baseDir := xl.AbsoluteFileDir(fileName)
	xl.Verbose("Reading file: %s, base dir: %s\n", fileName, baseDir)
	for {
		doc, err := docReader.ReadNextYamlDocument()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				reportFatalDocumentError(fileName, doc, err)
			}
		}
		imports = append(imports, extractImports(baseDir, doc)...)
		documents = append(documents, doc)
	}
	reader.Close()
	return FileWithDocuments{imports, documents, fileName}
}

func validateFileWithDocs(filesWithDocs []FileWithDocuments) {
	funk.ForEach(filesWithDocs, func(file FileWithDocuments) {
		funk.ForEach(file.documents, func(doc *xl.Document) {
			if doc.Kind == importSpecKind && doc.ApiVersion != yamlFormatVersion {
				xl.Fatal("unknown apiVersion for %s spec kind: %s\n", importSpecKind, doc.ApiVersion)
			}
		})
	})
}

func parseDocuments(fileNames []string, seenFiles mapset.Set) []FileWithDocuments {
	result := make([]FileWithDocuments, 0)
	for _, fileName := range fileNames {
		if !seenFiles.Contains(fileName) {
			fileWithDocuments := readDocumentsFromFile(fileName)
			result = append(result, fileWithDocuments)
			seenFiles.Add(fileName)
			result = append(parseDocuments(fileWithDocuments.imports, seenFiles), result...)
		}
	}
	validateFileWithDocs(result)
	return result
}

func DoApply(applyFilenames []string) {
	homeValsFiles, e := listHomeXlValsFiles()

	if e != nil {
		xl.Fatal("Error while reading value files from home: %s\n", e)
	}

	docs := parseDocuments(xl.ToAbsolutePaths(applyFilenames), mapset.NewSet())

	for _, fileWithDocs := range docs {
		projectValsFiles, err := listRelativeXlValsFiles(filepath.Dir(fileWithDocs.fileName))
		if err != nil {
			xl.Fatal("Error while reading value files for %s from project: %s\n", fileWithDocs.fileName, err)
		}

		allValsFiles := append(homeValsFiles, projectValsFiles...)

		context, err := xl.BuildContext(viper.GetViper(), &applyValues, allValsFiles)
		if err != nil {
			xl.Fatal("Error while reading configuration: %s\n", err)
		}
		if xl.IsVerbose {
			xl.Info("Context for document %s\n", fileWithDocs.fileName)
			context.PrintConfiguration()
		}

		xl.StartProgress(fileWithDocs.fileName)
		applyDir := filepath.Dir(fileWithDocs.fileName)

		for _, doc := range fileWithDocs.documents {
			xl.UpdateProgressStartDocument(fileWithDocs.fileName, doc)
			if doc.Kind != importSpecKind {
				changes, err := context.ProcessSingleDocument(doc, applyDir)
				printChanges(changes)
				if err != nil {
					reportFatalDocumentError(fileWithDocs.fileName, doc, err)
				}
			}
			xl.UpdateProgressEndDocument()
		}

		xl.EndProgress()
	}
}

var isFieldAlreadySetErrorRegexp = regexp.MustCompile(`field \w+ already set in type`)

func reportFatalDocumentError(applyFilename string, doc *xl.Document, err error) {
	if isFieldAlreadySetErrorRegexp.MatchString(err.Error()) {
		err = errors.Wrap(err, "Possible missing triple dash (---) to separate multiple YAML documents")
	}

	xl.Fatal("Error while processing YAML document at line %d of XL YAML file %s: %s\n", doc.Line, applyFilename, err)
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyFlags := applyCmd.Flags()
	applyFlags.StringArrayVarP(&applyFilenames, "file", "f", []string{}, "Path(s) to the file(s) to apply (required)")
	applyCmd.MarkFlagRequired("file")
	applyFlags.StringToStringVar(&applyValues, "values", map[string]string{}, "Values")
}

func listHomeXlValsFiles() ([]string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}
	xebialabsFolder := path.Join(home, ".xebialabs")
	if _, err := os.Stat(xebialabsFolder); os.IsNotExist(err) {
		return []string{}, nil
	}
	valfiles, err := xl.FindByExtInDirSorted(xebialabsFolder, ".xlvals")
	if err != nil {
		return nil, err
	}
	return valfiles, nil
}

func listRelativeXlValsFiles(dir string) ([]string, error) {
	valfiles, err := xl.FindByExtInDirSorted(dir, ".xlvals")
	if err != nil {
		return nil, err
	}
	return valfiles, nil
}
