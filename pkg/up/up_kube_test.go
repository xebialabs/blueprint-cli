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

func Test_processNodeList(t *testing.T) {
	var nodeList v1.NodeList
	var client kubernetes.Clientset
	var node v1.Node
	var nodes []v1.Node
	var address v1.NodeAddress
	var addresses []v1.NodeAddress
	node.Status.Addresses = append(addresses, address)
	nodeList.Items = append(nodes, node)

	test1Address := "127.0.0.2"
	test2Address := "127.0.0.2"
	nodeList.Items[0].Status.Addresses[0].Type = INTERNALIP
	nodeList.Items[0].Status.Addresses[0].Address = test1Address
	methodCalled := false
	getMinikubeEndpoint = func(client *kubernetes.Clientset) (s2 string, err error) {
		methodCalled = true
		return test2Address, nil
	}
	t.Run("given an internal IP return the internal ip", func(t *testing.T) {
		ip, err := processNodeList(&nodeList, &client)
		assert.Nil(t, err)
		assert.Equal(t, getURLWithPort(test1Address), ip)
		assert.False(t, methodCalled)
	})
	nodeList.Items[0].Status.Addresses[0].Type = HOSTNAME
	nodeList.Items[0].Status.Addresses[0].Address = MINIKUBE
	methodCalled = false
	t.Run("given an internal IP return the internal ip", func(t *testing.T) {
		ip, err := processNodeList(&nodeList, &client)
		assert.Nil(t, err)
		assert.Equal(t, test2Address, ip)
		assert.True(t, methodCalled)
	})
}

func Test_processServiceList(t *testing.T) {
	var client *kubernetes.Clientset
	var service v1.Service
	service.Spec.Type = LOADBALANCER
	var ingresses []v1.LoadBalancerIngress
	var ingress v1.LoadBalancerIngress
	const testHost = "testhost.com"
	ingress.Hostname = testHost
	ingress.IP = ""
	ingresses = append(ingresses, ingress)
	service.Status.LoadBalancer.Ingress = ingresses
	methodCalled := false
	getNodePortIp = func(client *kubernetes.Clientset) (s2 string, err error) {
		methodCalled = true
		return "127.0.0.2", nil
	}
	t.Run("given an ingress proxy and a loadbalancer, hostname is returned", func(t *testing.T) {
		ip, err, b := processServiceList(INGRESSPROXY, service, client)
		assert.Nil(t, err)
		assert.True(t, b)
		assert.Equal(t, getURLWithoutPort(testHost), ip)
		assert.False(t, methodCalled)
	})
	service.Status.LoadBalancer.Ingress[0].Hostname = ""
	const testIp = "127.0.0.1"
	service.Status.LoadBalancer.Ingress[0].IP = testIp
	methodCalled = false
	t.Run("given an ingress proxy and no loadbalancer, ip is returned", func(t *testing.T) {
		ip, err, b := processServiceList(INGRESSPROXY, service, client)
		assert.Nil(t, err)
		assert.True(t, b)
		assert.Equal(t, getURLWithoutPort(testIp), ip)
		assert.False(t, methodCalled)
	})
	methodCalled = false
	service.Spec.Type = NODEPORT
	service.Status.LoadBalancer.Ingress[0].Hostname = ""
	t.Run("given an internal XLD and a nodeport ", func(t *testing.T) {
		ip, err, b := processServiceList(XLINTERNAL, service, client)
		assert.Nil(t, err)
		assert.True(t, b)
		assert.Equal(t, "127.0.0.2", ip)
		assert.True(t, methodCalled)
	})
	methodCalled = false
	service.Spec.Type = LOADBALANCER
	service.Status.LoadBalancer.Ingress[0].Hostname = ""
	const test3Ip = "127.0.0.3"
	service.Spec.LoadBalancerIP = test3Ip
	t.Run("given an internal XLD and a nodeport ", func(t *testing.T) {
		ip, err, b := processServiceList(XLINTERNAL, service, client)
		assert.Nil(t, err)
		assert.True(t, b)
		assert.Equal(t, getURLWithoutPort(test3Ip), ip)
		assert.False(t, methodCalled)
	})
}
