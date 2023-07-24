package k8s

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var sampleKubeConfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: dGVzdCB0aGUgc2hpdCBvdXQgb2YgdGhpcw==
    server: https://test.hcp.eastus.azmk8s.io:443
  name: testCluster
- cluster:
    certificate-authority-data: dGVzdCB0aGUgc2hpdCBvdXQgb2YgdGhpcw==
    server: https://random.sk1.eu-west-1.eks.amazonaws.com
  name: arn:aws:eks:eu-west-1:random:cluster/xl-up-master
- cluster:
    certificate-authority-data: dGVzdCB0aGUgc2hpdCBvdXQgb2YgdGhpcw==
    server: https://minikube:8443
  name: minikubeCluster
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
    cluster: minikubeCluster
    user: minikube
  name: minikube
- context:
    cluster: testCluster
    namespace: test
    user: clusterUser_testCluster_testCluster
  name: testCluster
- context:
    cluster: arn:aws:eks:eu-west-1:random:cluster/xl-up-master
    user: clusterUser_testCluster_testCluster
  name: arn:aws:eks:eu-west-1:random:cluster/xl-up-master
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
- name: minikube
  user:
    client-certificate: /path/to/client.crt
    client-key: client.key
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
