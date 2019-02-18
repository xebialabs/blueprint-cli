package xl

import (
	"github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type FileWithDocuments struct {
	Imports   []string
	Parent    *string
	Documents []*Document
	FileName  string
}

func checkForEmptyImport(importedFile string) {
	if len(strings.TrimSpace(importedFile)) == 0 {
		util.Fatal("The 'imports' field contains empty elements.\n")
	}
}

func extractImports(baseDir string, doc *Document) []string {
	if doc.Metadata != nil && doc.Metadata["imports"] != nil {
		if imports, ok := doc.Metadata["imports"].([]interface{}); !ok {
			util.Fatal("The 'imports' field has wrong format. Must be a list of strings.\n")
		} else {
			delete(doc.Metadata, "imports")
			importedFiles, _ := funk.Map(imports, func(i interface{}) string {
				importedFile, _ := i.(string)
				checkForEmptyImport(importedFile)
				err := util.ValidateFilePath(importedFile, "imports")
				if err != nil {
					util.Fatal(err.Error())
				}
				return filepath.Join(baseDir, filepath.FromSlash(importedFile))
			}).([]string)
			return importedFiles
		}
	}
	return make([]string, 0)
}

var isFieldAlreadySetErrorRegexp = regexp.MustCompile(`field \w+ already set in type`)

func ReportFatalDocumentError(applyFilename string, doc *Document, err error) {
	if isFieldAlreadySetErrorRegexp.MatchString(err.Error()) {
		err = errors.Wrap(err, "Possible missing triple dash (---) to separate multiple YAML documents")
	}

	util.Fatal("Error while processing YAML document at line %d of XL YAML file %s: %s\n", doc.Line, applyFilename, err)
}

func validateFileWithDocs(filesWithDocs []FileWithDocuments) {
	funk.ForEach(filesWithDocs, func(file FileWithDocuments) {
		funk.ForEach(file.Documents, func(doc *Document) {
			if doc.Kind == models.ImportSpecKind && doc.ApiVersion != models.YamlFormatVersion {
				util.Fatal("unknown apiVersion for %s spec kind: %s\n", models.ImportSpecKind, doc.ApiVersion)
			}
		})
	})
}

func readDocumentsFromFile(fileName string, parent *string) FileWithDocuments {
	reader, err := os.Open(fileName)
	if err != nil {
		util.Fatal("Error while opening XL YAML file %s: %s\n", fileName, err)
	}
	imports := make([]string, 0)
	documents := make([]*Document, 0)
	docReader := NewDocumentReader(reader)
	baseDir := util.AbsoluteFileDir(fileName)
	util.Verbose("Reading file: %s, base dir: %s\n", fileName, baseDir)
	for {
		doc, err := docReader.ReadNextYamlDocument()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				ReportFatalDocumentError(fileName, doc, err)
			}
		}
		imports = append(imports, extractImports(baseDir, doc)...)
		documents = append(documents, doc)
	}
	_ = reader.Close()
	return FileWithDocuments{imports, parent, documents, fileName}
}

func parseDocuments(fileNames []string, seenFiles mapset.Set, parent *string) []FileWithDocuments {
	result := make([]FileWithDocuments, 0)
	for _, fileName := range fileNames {
		if !seenFiles.Contains(fileName) {
			fileWithDocuments := readDocumentsFromFile(fileName, parent)
			result = append(result, fileWithDocuments)
			seenFiles.Add(fileName)
			result = append(parseDocuments(fileWithDocuments.Imports, seenFiles, &fileName), result...)
		}
	}
	validateFileWithDocs(result)
	return result
}

type DocumentCallback func(*Context, FileWithDocuments, *Document)

func ForEachDocument(operationName string, fileNames []string, values map[string]string, fn DocumentCallback) {
	homeValsFiles, e := ListHomeXlValsFiles()

	if e != nil {
		util.Fatal("Error while reading value files from home: %s\n", e)
	}

	docs := parseDocuments(util.ToAbsolutePaths(fileNames), mapset.NewSet(), nil)
	util.VerboseSeparator()
	for _, fileWithDocs := range docs {
		var currentFile = util.PrintableFileName(fileWithDocs.FileName)
		if fileWithDocs.Parent != nil {
			var parentFile = util.PrintableFileName(*fileWithDocs.Parent)
			util.Info("%s %s (imported by %s)\n", operationName, currentFile, parentFile)
		} else {
			util.Info("%s %s\n", operationName, currentFile)
		}

		projectValsFiles, err := ListRelativeXlValsFiles(filepath.Dir(fileWithDocs.FileName))
		if err != nil {
			util.Fatal("Error while reading value files for %s from project: %s\n", fileWithDocs.FileName, err)
		}

		allValsFiles := append(homeValsFiles, projectValsFiles...)

		context, err := BuildContext(viper.GetViper(), &values, allValsFiles)
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}

		for _, doc := range fileWithDocs.Documents {
			util.Verbose("---\n")
			util.Verbose("%s document at line %d\n\n", operationName, doc.Line)
			if doc.Kind != models.ImportSpecKind {
				fn(context, fileWithDocs, doc)
			} else {
				util.Info("Done\n")
			}
		}
		util.VerboseSeparator()
	}
}
