package models

import (
	"fmt"
	"strings"
)

// Repository provider enum - used in blueprint repository configuration
const (
	ProviderMock       string = "mock"
	ProviderGitHub     string = "github"
	ProviderHttp       string = "http"
)
var RepoProviders = []string { ProviderMock, ProviderGitHub, ProviderHttp }

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

	DefaultBlueprintRepositoryName      = "XL Blueprints"
	DefaultBlueprintRepositoryUrl       = "https://dist.xebialabs.com/public/blueprints/"
	DefaultBlueprintRepositoryOwner     = ""
	DefaultBlueprintRepositoryToken     = ""
	DefaultBlueprintRepositoryBranch    = ""
	DefaultBlueprintRepositoryProvider  = ProviderHttp
)


const XldApiVersion = "xl-deploy/v1"
const XlrApiVersion = "xl-release/v1"