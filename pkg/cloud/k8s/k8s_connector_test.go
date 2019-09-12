package k8s

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("should get the default cluster id when the cluster name is not given", func(t *testing.T) {
		clusterMap := make(map[string]string)
		clusterMap["EksClusterName"] = ""

		clusterID := getClusterIDFromClusterName(clusterMap)

		assert.Equal(t, clusterID, "xl-up-master")
	})

	t.Run("should get the default cluster id when the cluster name is given", func(t *testing.T) {
		clusterMap := make(map[string]string)
		clusterMap["EksClusterName"] = "test-xl-cluster"

		clusterID := getClusterIDFromClusterName(clusterMap)

		assert.Equal(t, clusterID, "test-xl-cluster")
	})
}

func TestGetRequiredPropertyFromMap(t *testing.T) {
	t.Run("get required property from the map", func(t *testing.T) {
		answerMap := make(map[string]string)
		answerMap["one"] = "one"

		out, err := GetRequiredPropertyFromMap("one", answerMap)
		require.Nil(t, err)
		assert.Equal(t, out, "one")
	})
}
