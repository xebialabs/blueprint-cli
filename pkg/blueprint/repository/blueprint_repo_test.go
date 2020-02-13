package repository

import (
    "github.com/stretchr/testify/assert"
    "github.com/xebialabs/xl-blueprint/pkg/models"
    "net/url"
    "testing"
)

func TestGenerateBlueprintFileDefinition(t *testing.T) {
    testUrl, _ := url.Parse("https://github.com/xebialabs/test")

    blueprints := make(map[string]*models.BlueprintRemote)
    tests := []struct {
        name           string
        blueprintPath  string
        filename       string
        filePath       string
        parsedUrl      *url.URL
    }{
        {
            "should generate remote file for local repo",
            "test/local",
            "blueprint.yaml",
            ".",
            nil,
        },
        {
            "should generate remote file for github repo",
            "aws/test",
            "test.yaml.tmpl",
            "xebialabs/test",
            testUrl,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            remoteFile := GenerateBlueprintFileDefinition(blueprints, tt.blueprintPath, tt.filename, tt.filePath, tt.parsedUrl)
            assert.Contains(t, blueprints, tt.blueprintPath)
            assert.NotNil(t, remoteFile)
            assert.Equal(t, tt.filename, remoteFile.Filename)
            assert.Equal(t, tt.filePath, remoteFile.Path)
            assert.Equal(t, tt.parsedUrl, remoteFile.Url)
        })
    }
}

func TestCheckIfBlueprintDefinitionFile(t *testing.T) {
    tests := []struct {
        name           string
        filename       string
        expected       bool
    }{
        {"should validate blueprint def filename: blueprint.yaml", "blueprint.yaml", true},
        {"should validate blueprint def filename: blueprint.yml", "blueprint.yml", true},
        {"should validate blueprint def filename: Blueprint.YAML", "Blueprint.YAML", true},
        {"should not validate blueprint def filename: blueprint.txt", "blueprint.txt", false},
        {"should not validate blueprint def filename: blueprint", "blueprint", false},
        {"should not validate blueprint def filename: other.yaml", "other.yaml", false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.expected, CheckIfBlueprintDefinitionFile(tt.filename))
        })
    }
}
