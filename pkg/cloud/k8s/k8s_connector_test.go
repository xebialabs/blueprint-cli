package k8s

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
)

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
