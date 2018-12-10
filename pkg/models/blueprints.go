package models

const (
	BlueprintOutputDir   = "xebialabs"
	BlueprintFinalPrompt = "Confirm to generate blueprint files?"
)

// Function Result definition
type FnResult interface {
	GetResult(module string, attr string, index int) ([]string, error)
}
