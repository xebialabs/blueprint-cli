package models

const (
	YamlFormatVersion = "xl/v1"

	BlueprintYamlFormatV1 = "xl/v1" // Deprecated as of 9.0.0
	BlueprintYamlFormatV2 = "xl/v2"

	BlueprintSpecKind = "Blueprint"
)

var BlueprintYamlFormatSupportedVersions = []string{BlueprintYamlFormatV2, BlueprintYamlFormatV1}
