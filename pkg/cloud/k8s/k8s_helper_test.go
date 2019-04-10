package k8s

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/xebialabs/xl-cli/pkg/models"
)

var simpleSampleKubeConfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: 123==123
    insecure-skip-tls-verify: true
    server: https://test.io:443
  name: testCluster
contexts:
- context:
    cluster: testCluster
    namespace: test
    user: testCluster_user
  name: testCluster
current-context: testCluster
kind: Config
preferences: {}
users:
- name: testCluster_user
  user:
    client-certificate-data: 123==123
    client-key-data: 123==123
    token: 6555565666666666666`

var sampleKubeConfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: dGVzdCB0aGUgc2hpdCBvdXQgb2YgdGhpcw==
    server: https://test.hcp.eastus.azmk8s.io:443
  name: testCluster
- cluster:
    insecure-skip-tls-verify: true
    server: https://ocpm.test.com:8443
  name: ocpm-test-com:8443
- cluster:
    insecure-skip-tls-verify: true
    server: https://ocpm.test.com:8443
  name: testUserNotFound
contexts:
- context:
    cluster: ocpm-test-com:8443
    namespace: default
    user: test/ocpm-test-com:8443
  name: default/ocpm-test-com:8443/test
- context:
    cluster: testCluster
    namespace: test
    user: clusterUser_testCluster_testCluster
  name: testCluster
- context:
    cluster: testClusterNotFound
    namespace: test
    user: testClusterNotFound
  name: testClusterNotFound
- context:
    cluster: testUserNotFound
    namespace: test
    user: testUserNotFound
  name: testUserNotFound
current-context: testCluster
kind: Config
preferences: {}
users:
- name: clusterUser_testCluster_testCluster
  user:
    client-certificate-data: 123==123
    client-key-data: 123==123
    token: 6555565666666666666
- name: test/ocpm-test-com:8443
  user:
    client-certificate-data: 123==123
- name: testClusterNotFound
  user:
    client-certificate-data: 123==123`

func Setupk8sConfig() {
	tmpDir := filepath.Join("test", "blueprints")
	os.MkdirAll(tmpDir, os.ModePerm)
	d1 := []byte(sampleKubeConfig)
	ioutil.WriteFile(filepath.Join(tmpDir, "config"), d1, os.ModePerm)
	os.Setenv("KUBECONFIG", filepath.Join(tmpDir, "config"))
}

func TestGetKubeConfigFile(t *testing.T) {
	defer os.RemoveAll("test")
	tests := []struct {
		name    string
		want    []byte
		wantErr bool
		prepare func()
	}{
		{
			"should error if file not found",
			nil,
			true,
			func() {
				os.Setenv("KUBECONFIG", "test")
			},
		},
		{
			"should read file from path set as KUBECONFIG",
			[]byte(sampleKubeConfig),
			false,
			Setupk8sConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			got, err := GetKubeConfigFile()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKubeConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKubeConfigFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseKubeConfig(t *testing.T) {
	type args struct {
		kubeConfigYaml []byte
	}
	tests := []struct {
		name           string
		kubeConfigYaml []byte
		want           K8sConfig
		wantErr        bool
	}{
		{
			"should error on paring invalid config yaml",
			[]byte("----- gggg test}"),
			K8sConfig{},
			true,
		},
		{
			"should parse a valid config yaml",
			[]byte(simpleSampleKubeConfig),
			K8sConfig{
				APIVersion:     "v1",
				CurrentContext: "testCluster",
				Clusters: []K8sCluster{
					{
						Name: "testCluster",
						Cluster: K8sClusterItem{
							Server:                   "https://test.io:443",
							CertificateAuthorityData: "123==123",
							InsecureSkipTLSVerify:    true,
						},
					},
				},
				Contexts: []K8sContext{
					{
						Name: "testCluster",
						Context: K8sContextItem{
							Cluster:   "testCluster",
							Namespace: "test",
							User:      "testCluster_user",
						},
					},
				},
				Users: []K8sUser{
					{
						Name: "testCluster_user",
						User: K8sUserItem{
							ClientCertificateData: "123==123",
							ClientKeyData:         "123==123",
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKubeConfig(tt.kubeConfigYaml)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKubeConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseKubeConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetContext(t *testing.T) {
	config, _ := ParseKubeConfig([]byte(sampleKubeConfig))
	type args struct {
		config  K8sConfig
		context string
	}
	tests := []struct {
		name    string
		args    args
		want    K8SFnResult
		wantErr bool
	}{
		{
			"should error when context is not found",
			args{
				config:  config,
				context: "dummy",
			},
			K8SFnResult{},
			true,
		},
		{
			"should error when cluster is not found",
			args{
				config:  config,
				context: "testClusterNotFound",
			},
			K8SFnResult{},
			true,
		},
		{
			"should find default context when context is not specified",
			args{
				config:  config,
				context: "",
			},
			K8SFnResult{
				Cluster: K8sClusterItem{
					Server:                   "https://test.hcp.eastus.azmk8s.io:443",
					CertificateAuthorityData: "dGVzdCB0aGUgc2hpdCBvdXQgb2YgdGhpcw==",
					InsecureSkipTLSVerify:    false,
				},
				Context: K8sContextItem{
					Cluster:   "testCluster",
					Namespace: "test",
					User:      "clusterUser_testCluster_testCluster",
				},
				User: K8sUserItem{
					ClientCertificateData: "123==123",
					ClientKeyData:         "123==123",
				},
			},
			false,
		},
		{
			"should find specified context when context is specified",
			args{
				config:  config,
				context: "default/ocpm-test-com:8443/test",
			},
			K8SFnResult{
				Cluster: K8sClusterItem{
					Server:                "https://ocpm.test.com:8443",
					InsecureSkipTLSVerify: true,
				},
				Context: K8sContextItem{
					Cluster:   "ocpm-test-com:8443",
					Namespace: "default",
					User:      "test/ocpm-test-com:8443",
				},
				User: K8sUserItem{
					ClientCertificateData: "123==123",
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetContext(tt.args.config, tt.args.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetK8SConfigFromSystem(t *testing.T) {
	defer os.RemoveAll("test")
	Setupk8sConfig()

	tests := []struct {
		name    string
		context string
		want    K8SFnResult
		wantErr bool
	}{
		{
			"should error when context is not found",
			"dummy",
			K8SFnResult{},
			true,
		},
		{
			"should error when cluster is not found",
			"testClusterNotFound",
			K8SFnResult{},
			true,
		},
		{
			"should find default context when context is not specified",
			"",
			K8SFnResult{
				Cluster: K8sClusterItem{
					Server:                   "https://test.hcp.eastus.azmk8s.io:443",
					CertificateAuthorityData: "dGVzdCB0aGUgc2hpdCBvdXQgb2YgdGhpcw==",
					InsecureSkipTLSVerify:    false,
				},
				Context: K8sContextItem{
					Cluster:   "testCluster",
					Namespace: "test",
					User:      "clusterUser_testCluster_testCluster",
				},
				User: K8sUserItem{
					ClientCertificateData: "123==123",
					ClientKeyData:         "123==123",
				},
			},
			false,
		},
		{
			"should find specified context when context is specified",
			"default/ocpm-test-com:8443/test",
			K8SFnResult{
				Cluster: K8sClusterItem{
					Server:                "https://ocpm.test.com:8443",
					InsecureSkipTLSVerify: true,
				},
				Context: K8sContextItem{
					Cluster:   "ocpm-test-com:8443",
					Namespace: "default",
					User:      "test/ocpm-test-com:8443",
				},
				User: K8sUserItem{
					ClientCertificateData: "123==123",
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetK8SConfigFromSystem(tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetK8SConfigFromSystem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetK8SConfigFromSystem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCallK8SFuncByName(t *testing.T) {
	defer os.RemoveAll("test")
	Setupk8sConfig()

	type args struct {
		module string
		params []string
	}
	tests := []struct {
		name    string
		args    args
		want    models.FnResult
		wantErr bool
	}{
		{
			"should error when invalid module is specified",
			args{
				"k88s",
				[]string{""},
			},
			nil,
			true,
		},
		{
			"should return empty when invalid context is specified",
			args{
				"config",
				[]string{"test"},
			},
			&K8SFnResult{},
			false,
		},
		{
			"should fetch the k8s config with default context when valid module is specified",
			args{
				"cOnFiG", // to check case sensitivity
				[]string{""},
			},
			&K8SFnResult{
				Cluster: K8sClusterItem{
					Server:                   "https://test.hcp.eastus.azmk8s.io:443",
					CertificateAuthorityData: "dGVzdCB0aGUgc2hpdCBvdXQgb2YgdGhpcw==",
					InsecureSkipTLSVerify:    false,
				},
				Context: K8sContextItem{
					Cluster:   "testCluster",
					Namespace: "test",
					User:      "clusterUser_testCluster_testCluster",
				},
				User: K8sUserItem{
					ClientCertificateData: "123==123",
					ClientKeyData:         "123==123",
				},
			},
			false,
		},
		{
			"should fetch the k8s config with given context context when valid module is specified",
			args{
				"cOnFiG", // to check case sensitivity
				[]string{"default/ocpm-test-com:8443/test"},
			},
			&K8SFnResult{
				Cluster: K8sClusterItem{
					Server:                "https://ocpm.test.com:8443",
					InsecureSkipTLSVerify: true,
				},
				Context: K8sContextItem{
					Cluster:   "ocpm-test-com:8443",
					Namespace: "default",
					User:      "test/ocpm-test-com:8443",
				},
				User: K8sUserItem{
					ClientCertificateData: "123==123",
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CallK8SFuncByName(tt.args.module, tt.args.params...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CallK8SFuncByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CallK8SFuncByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getK8SConfigField(t *testing.T) {
	config, _ := ParseKubeConfig([]byte(sampleKubeConfig))
	fnRes, _ := GetContext(config, "testCluster")
	type args struct {
		v     *K8SFnResult
		field string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"should return empty when fetching non existing field",
			args{
				&fnRes,
				"dummy",
			},
			"",
		},
		{
			"should return value when fetching existing field",
			args{
				&fnRes,
				"cluster_server",
			},
			"https://test.hcp.eastus.azmk8s.io:443",
		},
		{
			"should return value when fetching existing field with snakecase",
			args{
				&fnRes,
				"cluster_insecure_skip_tls_verify",
			},
			"false",
		},
		{
			"should return value when fetching existing field with mix of camelcase and underscore",
			args{
				&fnRes,
				"cluster_insecureSkipTlsVerify",
			},
			"false",
		},
		{
			"should return value when fetching existing field with uneven case",
			args{
				&fnRes,
				"clusterInsecureSkipTlsVerify",
			},
			"false",
		},
		{
			"should return decoded value when fetching existing encoded field",
			args{
				&fnRes,
				"Cluster_CertificateAuthorityData",
			},
			`test the shit out of this`,
		},
		{
			"should return actual value when decoding fails",
			args{
				&fnRes,
				"User_clientCertificateData",
			},
			`123==123`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getK8SConfigField(tt.args.v, tt.args.field); got != tt.want {
				t.Errorf("getK8SConfigField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlattenFields(t *testing.T) {
	config, _ := ParseKubeConfig([]byte(sampleKubeConfig))
	fnRes, _ := GetContext(config, "testCluster")
	tests := []struct {
		name  string
		iface interface{}
		want  int
		keys  []string
	}{
		{
			"should flatten all the fields",
			fnRes,
			8,
			[]string{"User_ClientCertificateData", "Cluster_Server", "Cluster_CertificateAuthorityData", "Cluster_InsecureSkipTLSVerify", "Context_Namespace", "Context_User", "Context_Cluster", "User_ClientKeyData"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlattenFields(tt.iface)
			if len(got) != tt.want {
				t.Errorf("FlattenFields() length = %v, want %v", got, tt.want)
			}
			keys := make([]string, 0, len(got))
			for k := range got {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			sort.Strings(tt.keys)
			if !reflect.DeepEqual(keys, tt.keys) {
				t.Errorf("FlattenFields() keys = %v, want %v", keys, tt.keys)
			}
		})
	}
}

func TestK8SFnResult_GetResult(t *testing.T) {
	config, _ := ParseKubeConfig([]byte(sampleKubeConfig))
	fnRes, _ := GetContext(config, "testCluster")
	type args struct {
		module string
		attr   string
		index  int
	}
	tests := []struct {
		name    string
		fields  K8SFnResult
		args    args
		want    []string
		wantErr bool
	}{
		{
			"should error if attribute is not set",
			fnRes,
			args{
				Config,
				"",
				-1,
			},
			nil,
			true,
		},
		{
			"should error if invalid module is not set",
			fnRes,
			args{
				"dummy",
				"",
				-1,
			},
			nil,
			true,
		},
		{
			"should check if valid config is available",
			fnRes,
			args{
				Config,
				"IsAvailable",
				-1,
			},
			[]string{"true"},
			false,
		},
		{
			"should fetch valid config field value",
			fnRes,
			args{
				Config,
				"cluster_server",
				-1,
			},
			[]string{"https://test.hcp.eastus.azmk8s.io:443"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fields
			got, err := result.GetResult(tt.args.module, tt.args.attr, tt.args.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("K8SFnResult.GetResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("K8SFnResult.GetResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
