package aws

import (
	"fmt"
	"strconv"

	"github.com/xebialabs/yaml"

	"reflect"
	"strings"

	"github.com/xebialabs/xl-cli/pkg/models"
)

const (
	Config = "config"
)

type K8sConfig struct {
	ApiVersion     string       `yaml:"apiVersion,omitempty"`
	Clusters       []K8sCluster `yaml:"clusters,omitempty"`
	Contexts       []K8sContext `yaml:"contexts,omitempty"`
	CurrentContext string       `yaml:"current-context,omitempty"`
	Users          []K8sUser    `yaml:"users,omitempty"`
}

type K8sCluster struct {
	Name    string `yaml:"name,omitempty"`
	Cluster struct {
		Server                   string `yaml:"server,omitempty"`
		CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
		InsecureSkipTlsVerify    bool   `yaml:"insecure-skip-tls-verify,omitempty"`
	} `yaml:"cluster,omitempty"`
}

type K8sContext struct {
	Name    string `yaml:"name,omitempty"`
	Context struct {
		Cluster    string `yaml:"cluster,omitempty"`
		Namesapace string `yaml:"namesapace,omitempty"`
		User       string `yaml:"user,omitempty"`
	} `yaml:"context,omitempty"`
}

type K8sUser struct {
	Name string `yaml:"name,omitempty"`
	User struct {
		ClientCertificateData string `yaml:"client-certificate-data,omitempty"`
		ClientKeyData         string `yaml:"client-key-data,omitempty"`
	} `yaml:"user,omitempty"`
}

type K8SFnResult struct {
	cluster K8sCluster
	context K8sContext
	user    K8sUser
}

func (result *K8SFnResult) GetResult(module string, attr string, index int) ([]string, error) {
	switch module {
	case Config:
		if attr == "" {
			return nil, fmt.Errorf("required attribute is not set")
		}

		// if requested, do exists check
		if attr == "IsAvailable" { // todo add another check when user has auth-provider
			return []string{strconv.FormatBool(result.cluster.Cluster.Server != "" && result.user.User.ClientCertificateData != "")}, nil
		}

		// return attribute
		return []string{getK8SConfigField(result, attr)}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid Kubernetes module", module)
	}
}

func getK8SConfigField(v *K8SFnResult, field string) string {
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(field)
	return f.String()
}

// CallK8SFuncByName calls related K8S module function with parameters provided
func CallK8SFuncByName(module string, params ...string) (models.FnResult, error) {
	switch strings.ToLower(module) {
	case Config:
		context := ""
		if len(params) > 0 && params[0] != "" {
			context = params[0]
		}
		config, err := GetK8SConfigFromSystem(context)
		if err != nil {
			// handle K8S configuration errors gracefully
			return &K8SFnResult{}, nil
		}
		return &config, nil
	default:
		return nil, fmt.Errorf("%s is not a valid K8S module", module)
	}
}

// Utilities

// GetK8SCredentialsFromSystem fetches stored K8S access keys from file or env keys
func GetK8SConfigFromSystem(context string) (K8SFnResult, error) {
	// fetch k8s config yaml and parse
	kubeConfigYaml, err := GetKubeConfigFile()
	if err != nil {
		return K8SFnResult{}, err
	}
	result, err := ParseKubeConfig(kubeConfigYaml)
	if err != nil {
		return K8SFnResult{}, err
	}
	// TODO get requested context
	contextRes, err := GetContext(result, context)
	if err != nil {
		return K8SFnResult{}, err
	}
	return contextRes, nil
}

func GetKubeConfigFile() ([]byte, error) {
	// TODO check for KUBECONFIG in environment
	// if not set find path based on OS
	// read file from path and return string
	return nil, nil
}

func ParseKubeConfig(kubeConfigYaml []byte) (K8sConfig, error) {
	// parse yaml
	res := K8sConfig{}
	err := yaml.Unmarshal(kubeConfigYaml, &res)
	if err != nil {
		return res, err
	}
	return res, nil
}

func GetContext(config K8sConfig, context string) (K8SFnResult, error) {
	// TODO if context not given, get default
	return K8SFnResult{}, nil
}
