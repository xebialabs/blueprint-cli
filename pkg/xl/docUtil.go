package xl

import (
	"github.com/deckarep/golang-set"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	"github.com/xebialabs/xl-cli/pkg/models"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func ListHomeXlValsFiles() ([]string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}
	xebialabsFolder := path.Join(home, ".xebialabs")
	if _, err := os.Stat(xebialabsFolder); os.IsNotExist(err) {
		return []string{}, nil
	}
	valfiles, err := FindByExtInDirSorted(xebialabsFolder, ".xlvals")
	if err != nil {
		return nil, err
	}
	return valfiles, nil
}

func ListRelativeXlValsFiles(dir string) ([]string, error) {
	valfiles, err := FindByExtInDirSorted(dir, ".xlvals")
	if err != nil {
		return nil, err
	}
	return valfiles, nil
}

func checkForEmptyImport(importedFile string) {
	if len(strings.TrimSpace(importedFile)) == 0 {
		Fatal("The 'imports' field contains empty elements.\n")
	}
}

func extractImports(baseDir string, doc *Document) []string {
	if doc.Metadata != nil && doc.Metadata["imports"] != nil {
		if imports, ok := doc.Metadata["imports"].([]interface{}); !ok {
			Fatal("The 'imports' field has wrong format. Must be a list of strings.\n")
		} else {
			delete(doc.Metadata, "imports")
			importedFiles, _ := funk.Map(imports, func(i interface{}) string {
				importedFile, _ := i.(string)
				checkForEmptyImport(importedFile)
				err := ValidateFilePath(importedFile, "imports")
				if err != nil {
					Fatal(err.Error())
				}
				return filepath.Join(baseDir, filepath.FromSlash(importedFile))
			}).([]string)
			return importedFiles
		}
	}
	return make([]string, 0)
}

func readDocumentsFromFile(fileName string, parent *string) FileWithDocuments {
	reader, err := os.Open(fileName)
	if err != nil {
		Fatal("Error while opening XL YAML file %s: %s\n", fileName, err)
	}
	imports := make([]string, 0)
	documents := make([]*Document, 0)
	docReader := NewDocumentReader(reader)
	baseDir := AbsoluteFileDir(fileName)
	Verbose("Reading file: %s, base dir: %s\n", fileName, baseDir)
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
	return FileWithDocuments{imports, parent, documents, fileName}
}

func validateFileWithDocs(filesWithDocs []FileWithDocuments) {
	funk.ForEach(filesWithDocs, func(file FileWithDocuments) {
		funk.ForEach(file.documents, func(doc *Document) {
			if doc.Kind == models.ImportSpecKind && doc.ApiVersion != models.YamlFormatVersion {
				Fatal("unknown apiVersion for %s spec kind: %s\n", models.ImportSpecKind, doc.ApiVersion)
			}
		})
	})
}

func ParseDocuments(fileNames []string, seenFiles mapset.Set, parent *string) []FileWithDocuments {
	result := make([]FileWithDocuments, 0)
	for _, fileName := range fileNames {
		if !seenFiles.Contains(fileName) {
			fileWithDocuments := readDocumentsFromFile(fileName, parent)
			result = append(result, fileWithDocuments)
			seenFiles.Add(fileName)
			result = append(ParseDocuments(fileWithDocuments.imports, seenFiles, &fileName), result...)
		}
	}
	validateFileWithDocs(result)
	return result
}

var isFieldAlreadySetErrorRegexp = regexp.MustCompile(`field \w+ already set in type`)

func reportFatalDocumentError(applyFilename string, doc *Document, err error) {
	if isFieldAlreadySetErrorRegexp.MatchString(err.Error()) {
		err = errors.Wrap(err, "Possible missing triple dash (---) to separate multiple YAML documents")
	}

	Fatal("Error while processing YAML document at line %d of XL YAML file %s: %s\n", doc.Line, applyFilename, err)
}
