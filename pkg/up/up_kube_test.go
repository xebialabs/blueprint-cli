package up

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Test_getKubeConfigMap(t *testing.T) {

	tests := []struct {
		name      string
		answerMap map[string]string
		want      string
		wantErr   bool
		prepare   func()
	}{
		{
			"Should return empty when namespace is not available",
			map[string]string{
				"K8sApiServerURL": "localhost",
				"K8sToken":        "localhost",
			},
			"",
			false,
			func() {
				getK8sConfigMaps = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.ConfigMapList, error) {
					return &v1.ConfigMapList{}, nil
				}
				getK8sNamespaces = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.NamespaceList, error) {
					return &v1.NamespaceList{}, nil
				}
			},
		},
		{
			"Should return config map string when namespace is available",
			map[string]string{
				"K8sApiServerURL": "localhost",
				"K8sToken":        "localhost",
			},
			"map data",
			false,
			func() {
				getK8sConfigMaps = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.ConfigMapList, error) {
					return &v1.ConfigMapList{
						Items: []v1.ConfigMap{
							v1.ConfigMap{
								ObjectMeta: metav1.ObjectMeta{Name: ConfigMapName},
								Data: map[string]string{
									DataFile: "map data",
								},
							},
						},
					}, nil
				}
				getK8sNamespaces = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.NamespaceList, error) {
					return &v1.NamespaceList{
						Items: []v1.Namespace{
							v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: NAMESPACE}},
						},
					}, nil
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			got, err := getKubeConfigMap(tt.answerMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKubeConfigMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getKubeConfigMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkForNameSpace(t *testing.T) {
	getK8sNamespaces = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.NamespaceList, error) {
		return &v1.NamespaceList{
			Items: []v1.Namespace{
				v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: NAMESPACE}},
			},
		}, nil
	}
	type args struct {
		client    *kubernetes.Clientset
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"should return true when namespace found",
			args{
				nil,
				"xebialabs",
			},
			true,
			false,
		},
		{
			"should return false when namespace not found",
			args{
				nil,
				"somens",
			},
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkForNameSpace(tt.args.client, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkForNameSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkForNameSpace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getKubeClient(t *testing.T) {
	tests := []struct {
		name      string
		answerMap map[string]string
		wantErr   bool
	}{
		{
			"Should fail if required properties are missing",
			map[string]string{
				"K8sApiServerURL": "localhost",
			},
			true,
		},
		{
			"Should return a k8s client",
			map[string]string{
				"K8sApiServerURL": "localhost",
				"K8sToken":        "1234",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getKubeClient(tt.answerMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKubeClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.NotNil(t, got)
			}
		})
	}
}
