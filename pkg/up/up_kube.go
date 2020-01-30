package up

import (
	"fmt"
	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"
	"github.com/xebialabs/xl-cli/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// The client function pointers are kept outside so that it can be mocked
var getK8sConfigMaps = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.ConfigMapList, error) {
	return client.CoreV1().ConfigMaps(NAMESPACE).List(opts)
}
var getK8sNamespaces = func(client *kubernetes.Clientset, opts metav1.ListOptions) (*v1.NamespaceList, error) {
	return client.CoreV1().Namespaces().List(opts)
}

// Constants to use
const (
	NAMESPACE    = "xebialabs"
	INGRESSPROXY = "haproxy-ingress"
	LOADBALANCER = "loadbalancer"
	HTTP         = "http"
	XLINTERNAL   = "xebialabs-internal"
	NODEPORT     = "nodeport"
	INTERNALIP   = "internalip"
	HTTPPORT     = "30080"
	HOSTNAME     = "hostname"
	MINIKUBE     = "minikube"
	KUBERNETES   = "kubernetes"
)

var getKubeClient = func(answerMap map[string]string) (*kubernetes.Clientset, error) {
	config, err := k8s.GetK8sConfiguration(answerMap)
	if err != nil {
		return nil, err
	}
	util.Verbose("Got the configuration...\n")

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes client: %s", err)
	}
	return client, nil
}

func getKubeConfigMap(answerMap map[string]string) (string, error) {
	// Step 1 Get connection
	client, err := getKubeClient(answerMap)
	if err != nil {
		return "", err
	}
	// Step 2 Check for namespace
	isNamespaceAvailable, err := checkForNameSpace(client, NAMESPACE)
	if err != nil {
		return "", err
	}
	// Step 3 Check for version
	if isNamespaceAvailable {
		util.Verbose("the namespace %s is available...\n", NAMESPACE)

		cm, err := getK8sConfigMaps(client, metav1.ListOptions{})
		if err != nil {
			return "", err
		}

		var out string

		for _, c := range cm.Items {
			if c.Name == ConfigMapName {
				out = c.Data[DataFile]
			}
		}
		// Returning the data in the config map
		return out, nil
	}

	return "", nil
}

func checkForNameSpace(client *kubernetes.Clientset, namespace string) (bool, error) {

	ns, err := getK8sNamespaces(client, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, n := range ns.Items {
		if n.Name == namespace {
			return true, nil
		}
	}
	return false, nil
}

func getURLWithoutPort(address string) string {
	return HTTP + "://" + address
}

func getURLWithPort(address string) string {
	return HTTP + "://" + address + ":" + HTTPPORT
}

var GetIp = func(client *kubernetes.Clientset) (string, error) {
	namespacePresent, err := checkForNameSpace(client, NAMESPACE)
	if err != nil {
		return "", err
	}
	if namespacePresent {
		listOptions := metav1.ListOptions{}
		serviceList, err := client.CoreV1().Services(NAMESPACE).List(listOptions)
		if err != nil {
			return "", err
		}
		for _, service := range serviceList.Items {
			name := service.GetObjectMeta().GetName()
			ip, err2, done := processService(name, service, client)
			if done {
				return ip, err2
			}
		}
	}
	return "", fmt.Errorf("could not get the address of the cluster")
}

func processService(name string, service v1.Service, client *kubernetes.Clientset) (string, error, bool) {
	location := ""
	if IsEqualIgnoreCase(name, INGRESSPROXY) && IsEqualIgnoreCase(string(service.Spec.Type), LOADBALANCER) {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.Hostname != "" {
				location = getURLWithoutPort(ingress.Hostname)
				return location, nil, true
			}
			location = getURLWithoutPort(ingress.IP)
			return location, nil, true
		}
	} else if IsEqualIgnoreCase(name, XLINTERNAL) {
		if IsEqualIgnoreCase(string(service.Spec.Type), NODEPORT) {
			ip, err := getNodePortIp(client)
			return ip, err, true
		} else if IsEqualIgnoreCase(string(service.Spec.Type), LOADBALANCER) && service.Spec.LoadBalancerIP != "" {
			location = getURLWithoutPort(service.Spec.LoadBalancerIP)
			return location, nil, true
		}
	}
	return "", nil, false
}

func getNodePortIp(client *kubernetes.Clientset) (string, error) {
	returnIp := ""
	listOptions := metav1.ListOptions{}

	nodeList, err := client.CoreV1().Nodes().List(listOptions)
	if err != nil {
		return "", err
	}
	returnIp, s2, err2 := processNodeList(nodeList, returnIp, err, client)
	if err2 != nil {
		return s2, err2
	}
	if returnIp != "" {
		return returnIp, nil
	}
	return "", fmt.Errorf("unable to get nodeport ip")
}

func processNodeList(nodeList *v1.NodeList, returnIp string, err error, client *kubernetes.Clientset) (string, string, error) {
	for _, node := range nodeList.Items {
		for _, address := range node.Status.Addresses {
			if IsEqualIgnoreCase(string(address.Type), INTERNALIP) {
				ip := address.Address
				if ip != "" {
					returnIp = getURLWithPort(ip)
				}
			}
			if IsEqualIgnoreCase(string(address.Type), HOSTNAME) && IsEqualIgnoreCase(address.Address, MINIKUBE) {
				returnIp, err = getMinikubeEndpoint(client)
				if err != nil {
					return "", "", err
				}
			}
		}
	}
	return returnIp, "", nil
}

func getMinikubeEndpoint(client *kubernetes.Clientset) (string, error) {
	listOptions := metav1.ListOptions{}
	endpointList, err := client.CoreV1().Endpoints(NAMESPACE).List(listOptions)
	if err != nil {
		return "", err
	}
	s2, err2, done := processEndpointList(endpointList)
	if done {
		return s2, err2
	}
	return "", fmt.Errorf("unable to get minikube endpoint")
}

func processEndpointList(endpointList *v1.EndpointsList) (string, error, bool) {
	for _, endpoint := range endpointList.Items {
		if IsEqualIgnoreCase(endpoint.GetObjectMeta().GetName(), KUBERNETES) {
			for _, subset := range endpoint.Subsets {
				for _, address := range subset.Addresses {
					if address.IP != "" {
						return getURLWithPort(address.IP), nil, true
					}
				}
			}
		}
	}
	return "", nil, false
}
