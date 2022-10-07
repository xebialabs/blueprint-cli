package models

import "net/url"

const (
	BlueprintOutputDir   = "xebialabs"
	BlueprintFinalPrompt = "Confirm to generate blueprint files?"
	UpFinalPrompt        = "Do you want to proceed to the deployment with these values?"
)

// Function Result definition
type FnResult interface {
	GetResult(module string, attr string, index int) ([]string, error)
}

// Blueprint Remote Definition
type RemoteFile struct {
	Filename string
	Path     string
	Url      *url.URL
}
type BlueprintRemote struct {
	Name           string
	Path           string
	DefinitionFile RemoteFile
	Files          []RemoteFile
}

func NewBlueprintRemote(name string, path string) *BlueprintRemote {
	r := new(BlueprintRemote)
	r.Name = name
	r.Path = path
	return r
}

func (blueprint *BlueprintRemote) AddFile(file RemoteFile) []RemoteFile {
	blueprint.Files = append(blueprint.Files, file)
	return blueprint.Files
}
