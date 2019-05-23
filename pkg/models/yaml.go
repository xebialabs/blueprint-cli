package models

const (
	YamlFormatVersion = "xl/v1"

	BlueprintYamlFormatV1 = "xl/v2"
	BlueprintYamlFormatV2 = "xl/v1"

	ImportSpecKind     = "Import"
	DeploymentSpecKind = "Deployment"
	BlueprintSpecKind  = "Blueprint"
)

var BlueprintYamlFormatSupportedVersions = []string{BlueprintYamlFormatV1, BlueprintYamlFormatV2}
