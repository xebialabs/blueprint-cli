package xl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
)

type FileWithDocuments struct {
	Imports   []string
	Parent    *string
	Documents []*Document
	FileName  string
	SCMInfo   *SCMInfo
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

	util.Fatal(
		"%sError while processing YAML document at line %d of XL YAML file %s:\n%s%s\n",
		util.Indent1(), doc.Line, applyFilename, util.Indent1(), err,
	)
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

func readDocumentsFromFile(fileName string, parent *string, process ToProcess, info *SCMInfo) FileWithDocuments {
	reader, err := os.Open(fileName)
	if err != nil {
		util.Fatal("Error while opening XL YAML file %s:\n%s\n", fileName, err)
	}
	imports := make([]string, 0)
	documents := make([]*Document, 0)
	docReader := NewDocumentReader(reader)
	baseDir := util.AbsoluteFileDir(fileName)
	for {
		doc, err := docReader.ReadNextYamlDocumentWithProcess(process)
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
	return FileWithDocuments{imports, parent, documents, fileName, info}
}

func ParseDocuments(fileNames []string, seenFiles mapset.Set, parent *string, process ToProcess, includeSCMInfo bool, skipDirtyCheck bool, cachedSCMInfo SCMInfo) []FileWithDocuments {
	result := make([]FileWithDocuments, 0)
	for _, fileName := range fileNames {
		if !seenFiles.Contains(fileName) {
            var scmInfo SCMInfo
            if cachedSCMInfo == (SCMInfo{}) {
                scmInfo = getSCMInfo(fileName, includeSCMInfo, skipDirtyCheck)
            } else {
                scmInfo = cachedSCMInfo
                scmInfo.filename = getRelativePath(fileName, cachedSCMInfo.localPath)
            }

			fileWithDocuments := readDocumentsFromFile(fileName, parent, process, &scmInfo)
			result = append(result, fileWithDocuments)
			seenFiles.Add(fileName)
			result = append(ParseDocuments(fileWithDocuments.Imports, seenFiles, &fileName, process, includeSCMInfo, skipDirtyCheck, scmInfo), result...)
		}
	}
	validateFileWithDocs(result)
	return result
}

type DocumentCallback func(*Context, FileWithDocuments, *Document)

func logOrFail(includeSCMInfo bool, err error, format string, a ...interface{}) {
	if err != nil {
		if includeSCMInfo {
			util.Fatal(format, a...)
		} else {
			util.Verbose("Ignoring SCM error: "+format, a...)
		}
	}
}

func getSCMInfo(filename string, includeSCMInfo bool, skipDirtyCheck bool) SCMInfo {
	var scmInfo SCMInfo
	if includeSCMInfo {
		util.Verbose("getting scm info for %s \n", filename)
		repo, err := FindRepo(filename)
		logOrFail(includeSCMInfo, err, "Error while opening SCM for directory %s: %s.\n", filename, err)
		if repo != nil {
		    var isDirty = false
		    if !skipDirtyCheck {
                util.Verbose("Checking if repository is dirty (this might take a while on large repositories)...\n")
                isDirty, err = repo.IsDirty()
                if err != nil {
                    util.Fatal("Unable to determine if repo is dirty: %s \n", err)
                }
                if isDirty {
                    util.Fatal("Repository dirty and SCM info is required. Please commit all untracked and modified files before applying or use the --proceed-when-dirty flag to skip dirty checking. Aborting. \n")
                } else {
                    util.Verbose("Repository clean\n")
                }
            }

            commitInfo, err := repo.LatestCommitInfo()

            logOrFail(includeSCMInfo, err, "Error while getting commit info: %s\n", err)

            relativeFilename := getRelativePath(filename, repo.LocalPath())

            remote, err := repo.Remote()

            scmInfo = SCMInfo{relativeFilename, repo.SCM(), remote,
                commitInfo.Commit, commitInfo.Author, commitInfo.Date, commitInfo.Message, repo.LocalPath()}

            util.Verbose("Detected SCM Info: %s - dirty %t - %s - %s - %s - %s - %s - %s \n", repo.SCM(), isDirty, remote, relativeFilename, commitInfo.Commit, commitInfo.Author, commitInfo.Date, commitInfo.Message)
		}
	}
	return scmInfo
}

func getRelativePath(fullPath string, relativePath string) string {
    runes := []rune(fullPath)
    return string(runes[len(relativePath)+1:])
}

func ForEachDocument(operationName string, fileNames []string, values map[string]string, includeSCMInfo bool, skipDirtyCheck bool, fn DocumentCallback) {
	homeValsFiles, e := ListHomeXlValsFiles()

	if e != nil {
		util.Fatal("Error while reading value files from home: %s\n", e)
	}

	absolutePaths := util.ToAbsolutePaths(fileNames)
	// parsing
	docs := ParseDocuments(absolutePaths, mapset.NewSet(), nil, ToProcess{true, true, true}, includeSCMInfo, skipDirtyCheck, SCMInfo{})
	for fileIdx, fileWithDocs := range docs {
		var currentFile = util.PrintableFileName(fileWithDocs.FileName)
		progress := fmt.Sprintf("[%d/%d]", fileIdx+1, len(docs))

		if fileWithDocs.Parent != nil {
			var parentFile = util.PrintableFileName(*fileWithDocs.Parent)
			util.Info("%s %s %s (imported by %s)\n", progress, operationName, currentFile, parentFile)
		} else {
			util.Info("%s %s %s\n", progress, operationName, currentFile)
		}

		projectValsFiles, err := ListRelativeXlValsFiles(filepath.Dir(fileWithDocs.FileName))
		if err != nil {
			util.Fatal("Error while reading value files for %s from project: %s\n", fileWithDocs.FileName, err)
		}

		allValsFiles := append(homeValsFiles, projectValsFiles...)

		context, err := BuildContext(viper.GetViper(), &values, allValsFiles, fileWithDocs.SCMInfo, "")
		if err != nil {
			util.Fatal("Error while reading configuration: %s\n", err)
		}

		for docIdx, doc := range fileWithDocs.Documents {
			util.Verbose("%s%s document at line %d\n", util.Indent1(), operationName, doc.Line)
			if doc.Kind != models.ImportSpecKind {
				fn(context, fileWithDocs, doc)
			} else {
				util.Info("Done\n")
			}
			if docIdx < len(fileWithDocs.Documents)-1 {
				util.Verbose("\n")
			}
		}
		if fileIdx < len(docs)-1 {
			util.Info("\n")
		}
	}
}
