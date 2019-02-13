package models

import (
	"fmt"
	"strings"
)

// Repository provider enum - used in blueprint repository configuration
const (
	ProviderMock       string = "mock"
	ProviderGitHub     string = "github"
)
var RepoProviders = []string { ProviderMock, ProviderGitHub }

func GetRepoProvider(s string) (string, error) {
	for _, repoProvider := range RepoProviders {
		if repoProvider == strings.ToLower(s) {
			return repoProvider, nil
		}
	}
	return "", fmt.Errorf("%s is not supported as repository provider", s)
}

const (
	DefaultXlDeployUrl                  = "http://localhost:4516/"
	DefaultXlDeployUsername             = "admin"
	DefaultXlDeployPassword             = "admin"

	DefaultXlReleaseUrl                 = "http://localhost:5516/"
	DefaultXlReleaseUsername            = "admin"
	DefaultXlReleasePassword            = "admin"

	DefaultBlueprintRepositoryName      = "blueprints"
	DefaultBlueprintRepositoryOwner     = "xebialabs"
	DefaultBlueprintRepositoryToken     = ""
	DefaultBlueprintRepositoryBranch    = "master"
	DefaultBlueprintRepositoryProvider  = ProviderGitHub
)


const XldApiVersion = "xl-deploy/v1"
const XlrApiVersion = "xl-release/v1"