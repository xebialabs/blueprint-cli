package up

import (
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"
	"github.com/xebialabs/xl-cli/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// The namespace to use
const NAMESPACE = "xebialabs"

func getKubeClient() *kubernetes.Clientset {
	answerMap, err := blueprint.GetValuesFromAnswersFile(GeneratedAnswerFile)
	if err != nil {
		util.Fatal("Error reading answer file: %s\n", err.Error())
	}

	config := k8s.GetK8sConfiguration(answerMap)

	util.Verbose("Got the configuration...\n")

	client, err := kubernetes.NewForConfig(config)

	if err != nil {
		util.Fatal("Error creating kubernetes client: %s\n", err.Error())
	}
	return client
}

func getKubeConfigMap() string {
	// Step 1 Get connection
	client := getKubeClient()

	// Step 2 Check for namespace
	isNamespaceAvailable := checkForNameSpace(client, NAMESPACE)

	if isNamespaceAvailable {
		util.Verbose("the namespace %s is available...\n", NAMESPACE)

		cm, err := client.CoreV1().ConfigMaps(NAMESPACE).List(metav1.ListOptions{})
		if err != nil {
			util.Fatal(err.Error())
		}

		var out string

		for _, c := range cm.Items {
			if c.Name == ConfigMapName {
				out = c.Data[DataFile]
			}
		}
		// Returning the data in the config map
		return out
	}

	return ""
}

func checkForNameSpace(client *kubernetes.Clientset, namespace string) bool {

	ns, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		util.Fatal("Error while fetching namespaces: %s\n", err.Error())
	}

	for _, n := range ns.Items {
		if n.Name == namespace {
			return true
		}
	}
	return false
}
