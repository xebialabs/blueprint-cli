package up

import (
	"fmt"

	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"
	"github.com/xebialabs/xl-cli/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// The namespace to use
const NAMESPACE = "xebialabs"

func getKubeClient(answerMap map[string]string) (*kubernetes.Clientset, error) {
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

		cm, err := client.CoreV1().ConfigMaps(NAMESPACE).List(metav1.ListOptions{})
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

	ns, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
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
