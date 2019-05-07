package k8s

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestIsPropertyPresent(t *testing.T) {
	t.Run("should return true when the property is present", func(t *testing.T) {

		someMap := make(map[string]string)
		someMap["one"] = "one"
		someMap["two"] = "two"

		assert.Equal(t, IsPropertyPresent("one", someMap), true)
		assert.Equal(t, IsPropertyPresent("two", someMap), true)
	})

	t.Run("should return false when the property is not present", func(t *testing.T) {

		someMap := make(map[string]string)
		someMap["one"] = "one"
		someMap["two"] = "two"

		assert.Equal(t, IsPropertyPresent("three", someMap), false)
		assert.Equal(t, IsPropertyPresent("four", someMap), false)
	})

	t.Run("should return false when the property is not present and map is empty", func(t *testing.T) {

		someMap := make(map[string]string)

		assert.Equal(t, IsPropertyPresent("one", someMap), false)
		assert.Equal(t, IsPropertyPresent("two", someMap), false)
	})
}

func TestClusterIDorDefaultCluster(t *testing.T) {
	t.Run("should get the cluster id when the cluster name is given", func(t *testing.T) {
		clusterMap := make(map[string]string)
		clusterMap["eksClusterName"] = "arn:aws:6d996843a19cba481fc798705119203b.sk1.eu-west-1.eks.amazonaws.com/some-random-cluster-id"

		clusterID := getClusterIDFromClusterName(clusterMap)

		assert.Equal(t, clusterID, "some-random-cluster-id")
	})

	t.Run("should get the default cluster id when the cluster name is not given", func(t *testing.T) {
		clusterMap := make(map[string]string)
		clusterMap["eksClusterName"] = ""

		clusterID := getClusterIDFromClusterName(clusterMap)

		assert.Equal(t, clusterID, "xl-up-master")
	})

	t.Run("should get the default cluster id when the cluster name is not correct", func(t *testing.T) {
		clusterMap := make(map[string]string)
		clusterMap["eksClusterName"] = "https://6d996843a19cba481fc798705119203b.sk1.eu-west-1.eks.amazonaws.com/"

		clusterID := getClusterIDFromClusterName(clusterMap)

		assert.Equal(t, clusterID, "xl-up-master")
	})
}

func TestGetRequiredPropertyFromMap(t *testing.T) {
	t.Run("get required property from the map", func(t *testing.T) {
		answerMap := make(map[string]string)
		answerMap["one"] = "one"

		assert.Equal(t, GetRequiredPropertyFromMap("one", answerMap), "one")
	})
}
