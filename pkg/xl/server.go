package xl

import (
	"fmt"
	"github.com/thoas/go-funk"
)

const XldApiVersion = "xl-deploy/v1"
const XlrApiVersion = "xl-release/v1"

type XLServer interface {
	AcceptsDoc(doc *Document) bool
	PreprocessDoc(doc *Document)
	SendDoc(doc *Document) (*Changes, error)
	GetTaskStatus(taskId string) (*TaskState, error)
	GenerateDoc(filename string, path string, override bool) error
}

type XLDeployServer struct {
	Server             HTTPServer
	ApplicationsHome   string
	ConfigurationHome  string
	EnvironmentsHome   string
	InfrastructureHome string
}

type XLReleaseServer struct {
	Server HTTPServer
	Home   string
}

func (server *XLDeployServer) AcceptsDoc(doc *Document) bool {
	return doc.ApiVersion == XldApiVersion
}

func (server *XLReleaseServer) AcceptsDoc(doc *Document) bool {
	return doc.ApiVersion == XlrApiVersion
}

func (server *XLDeployServer) PreprocessDoc(doc *Document) {
	addHomeIfMissing(doc, server.ApplicationsHome, "Applications-home")
	addHomeIfMissing(doc, server.EnvironmentsHome, "Environments-home")
	addHomeIfMissing(doc, server.InfrastructureHome, "Infrastructure-home")
	addHomeIfMissing(doc, server.ConfigurationHome, "Configuration-home")
}

func (server *XLReleaseServer) PreprocessDoc(doc *Document) {
	addHomeIfMissing(doc, server.Home, "home")
}

func addHomeIfMissing(doc *Document, home string, key string) {
	if _, found := doc.Metadata[key]; home != "" && !found {
		doc.Metadata[key] = home
	}
}

func (server *XLDeployServer) GenerateDoc(filename string, path string, override bool) error {
	return server.Server.GenerateYamlDoc(filename, "deployit/devops-as-code/generate?path="+path, override)
}

func (server *XLReleaseServer) GenerateDoc(filename string, path string, override bool) error {
	return server.Server.GenerateYamlDoc(filename, "devops-as-code/generate?path="+path, override)
}

func (server *XLDeployServer) SendDoc(doc *Document) (*Changes, error) {
	return sendDoc(server.Server, "deployit/devops-as-code/apply", doc)
}

func (server *XLReleaseServer) SendDoc(doc *Document) (*Changes, error) {
	if doc.ApplyZip != "" {
		return nil, fmt.Errorf("file tags found but XL Release does not support file references")
	}
	return sendDoc(server.Server, "devops-as-code/apply", doc)
}

func findCurrentSteps(activeBlocks []interface{}, root []interface{}) []CurrentStep {
	result := make([]CurrentStep, 0)
	for _, phaseOrBlock := range root {
		var currentBlock = phaseOrBlock.(map[string]interface{})
		if isPhase, phasePropertyExists := currentBlock["phase"]; phasePropertyExists && isPhase.(string) == "true" {
			currentBlock = currentBlock["block"].(map[string]interface{})
		}

		if funk.Contains(activeBlocks, currentBlock["id"]) {
			result = append(result, CurrentStep{
				Name:      currentBlock["description"].(string),
				State:     currentBlock["state"].(string),
				Automated: true,
			})
		}
		if internalBlocks, internalBlocksOk := currentBlock["blocks"]; internalBlocksOk {
			internalResult := findCurrentSteps(activeBlocks, internalBlocks.([]interface{}))
			result = append(result, internalResult...)
		}
	}
	return result
}

func (server *XLDeployServer) GetTaskStatus(taskId string) (*TaskState, error) {
	js, err := server.Server.TaskInfo("deployit/tasks/v2/" + taskId)
	if err != nil {
		return nil, err
	}

	var currentSteps = make([]CurrentStep, 0)
	var activeBlocks = make([]interface{}, 0)
	if currentActiveBlocks, hasActiveBlocks := js["activeBlocks"]; hasActiveBlocks {
		activeBlocks = currentActiveBlocks.([]interface{})
	}

	if block, blockOk := js["block"]; blockOk {
		if blocks, blocksOk := block.(map[string]interface{})["blocks"]; blocksOk {
			currentSteps = findCurrentSteps(activeBlocks, blocks.([]interface{}))
		}
	}

	return &TaskState{
		State:        js["state"].(string),
		CurrentSteps: currentSteps,
	}, nil
}

func (server *XLReleaseServer) GetTaskStatus(taskId string) (*TaskState, error) {
	js, err := server.Server.TaskInfo("releases/" + taskId)
	if err != nil {
		return nil, err
	}
	steps := make([]CurrentStep, 0)

	if currentSimpleTasks, tasksExists := js["currentSimpleTasks"].([]interface{}); tasksExists {
		for _, task := range currentSimpleTasks {
			currentTask := task.(map[string]interface{})
			steps = append(steps, CurrentStep{
				Name:      currentTask["title"].(string),
				State:     currentTask["status"].(string),
				Automated: currentTask["automated"].(bool),
			})
		}
	}

	return &TaskState{State: js["status"].(string), CurrentSteps: steps}, nil
}

func sendDoc(server HTTPServer, path string, doc *Document) (*Changes, error) {
	if doc.ApplyZip != "" {
		Verbose("\tdocument contains !file tags, sending ZIP file with YAML document and artifacts to server\n")
		return server.PostYamlZip(path, doc.ApplyZip)
	} else {
		documentBytes, err := doc.RenderYamlDocument()
		if err != nil {
			return nil, err
		}
		return server.PostYamlDoc(path, documentBytes)
	}
}
