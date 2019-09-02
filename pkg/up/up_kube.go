package up

import (
	"github.com/xebialabs/xl-cli/pkg/blueprint"
	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"
	"github.com/xebialabs/xl-cli/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// The namespace to use
const NAMESPACE = "xebialabs"

func connectToKube() string {
	answerMap, err := blueprint.GetValuesFromAnswersFile(GeneratedAnswerFile)
	if err != nil {
		util.Fatal(err.Error())
	}

	// Step 1 Check connection
	config := k8s.GetK8sConfiguration(answerMap)
	util.Verbose("Got the configuration...\n")

	// Step 2 Check for namespace
	isNamespaceAvailable := checkForNameSpace(config, NAMESPACE)
	if isNamespaceAvailable {
		util.Verbose("the namespace %s is available...\n", NAMESPACE)
	}

	// Step 3 Check for version
	if isNamespaceAvailable {
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		cm, err := client.CoreV1().ConfigMaps(NAMESPACE).List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
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

func checkForNameSpace(config *restclient.Config, namespace string) bool {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	ns, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, n := range ns.Items {
		if n.Name == namespace {
			return true
		}
	}
	return false
}
