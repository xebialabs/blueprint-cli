package up

import (
	"fmt"
	"time"

	"github.com/xebialabs/xl-cli/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const LabelSelector = "organization=xebialabs"

func undeployAll(client *kubernetes.Clientset) error {
	if err := undeployNamespace(client); err != nil {
		return err
	}

	if err := undeployStorageClasses(client); err != nil {
		return err
	}

	if err := undeployClusterRoleBindings(client); err != nil {
		return err
	}

	if err := undeployClusterRoles(client); err != nil {
		return err
	}

	return nil
}

func undeployNamespace(client *kubernetes.Clientset) error {
	deletePolicy := metav1.DeletePropagationForeground

	util.Info("Deleting namespace...\n")

	err := client.CoreV1().Namespaces().Delete(NAMESPACE, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	if err != nil {
		if err.Error() != fmt.Sprintf("namespaces \"%s\" not found", NAMESPACE) {
			return fmt.Errorf("an error occurred - %s", err)
		} else {
			util.Info("Namespace \"%s\" was not found - continuing undeployment of other resources\n", NAMESPACE)
			return nil
		}
	}

	err = waitForUndeployCompletion(func() (int, error) {
		_, err := client.CoreV1().Namespaces().Get(NAMESPACE, metav1.GetOptions{})

		if err != nil {
			return 0, err
		} else {
			return 1, nil
		}
	}, "Namespace")

	if err != nil {
		return err
	}

	util.Info("Namespace deleted\n")

	return nil
}

func undeployStorageClasses(client *kubernetes.Clientset) error {
	// Delete storage classes matching label "xebialabs"
	util.Info("Deleting storage classes...\n")

	err := client.StorageV1().StorageClasses().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: LabelSelector,
	})

	if err != nil {
		return fmt.Errorf("an error occurred - %s", err)
	}

	err = waitForUndeployCompletion(func() (int, error) {
		items, err := client.StorageV1().StorageClasses().List(metav1.ListOptions{
			LabelSelector: LabelSelector,
		})

		if err != nil {
			return 0, err
		}

		return len(items.Items), nil
	}, "StorageClasses")

	if err != nil {
		return err
	}

	util.Info("Storage classes deleted\n")

	return nil
}

func undeployClusterRoleBindings(client *kubernetes.Clientset) error {
	// Delete cluster role bindings matching label "xebialabs"
	util.Info("Deleting cluster role bindings...\n")

	err := client.RbacV1().ClusterRoleBindings().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: LabelSelector,
	})

	if err != nil {
		return fmt.Errorf("an error occurred - %s", err)
	}

	err = waitForUndeployCompletion(func() (int, error) {
		items, err := client.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{
			LabelSelector: LabelSelector,
		})

		if err != nil {
			return 0, err
		}

		return len(items.Items), nil
	}, "ClusterRoleBindings")

	if err != nil {
		return err
	}

	util.Info("Cluster role bindings deleted\n")

	return nil
}

func undeployClusterRoles(client *kubernetes.Clientset) error {
	// Delete cluster roles matching label "xebialabs"
	util.Info("Deleting cluster roles...\n")

	err := client.RbacV1().ClusterRoles().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: LabelSelector,
	})

	if err != nil {
		return fmt.Errorf("an error occurred - %s", err)
	}

	err = waitForUndeployCompletion(func() (int, error) {
		items, err := client.RbacV1().ClusterRoles().List(metav1.ListOptions{
			LabelSelector: LabelSelector,
		})

		if err != nil {
			return 0, err
		}

		return len(items.Items), nil
	}, "ClusterRoles")

	if err != nil {
		return err
	}

	util.Info("Cluster roles deleted\n")

	return nil
}

type GetStatusFn func() (int, error)

func waitForUndeployCompletion(getStatusFn GetStatusFn, resource string) error {
	iterations := 0

	for {
		size, err := getStatusFn()

		if err != nil || size == 0 {
			break
		} else if iterations == 50 {
			return fmt.Errorf("reached 50 iterations while waiting for resource \"%s\" to delete. Failing now", resource)
		} else {
			util.Info("\tResource \"%s\" still deleting; Sleeping for 5 seconds\n", resource)
			iterations++
			time.Sleep(5 * time.Second)
		}
	}

	util.Info("\tResource \"%s\" deleted\n", resource)

	return nil
}
