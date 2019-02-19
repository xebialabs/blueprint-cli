package cmd

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"os"
	"path/filepath"
	"testing"
)

func TestPreview(t *testing.T) {
	util.IsVerbose = true

	t.Run("should preview multiple yaml files value replacement to xld", func(t *testing.T) {
		tempDir1 := createTempDir("firstDir")
		writeToFile(filepath.Join(tempDir1, "prop1.xlvals"), "env1=Environments/env1\nenv2=Environments/env2")
		yaml := writeToTempFile(tempDir1, "yaml1", fmt.Sprintf(`
apiVersion: %s
kind: Deployment
spec:
  package: Applications/PetPortal/1.0
  environment: !value env1
  undeployDependencies: true
  orchestrators:
    - parallel-by-deployed

---
apiVersion: %s
kind: Deployment
spec:
  package: Applications/PetPortal/2.0
  environment: !value env2
`, xl.XldApiVersion, xl.XldApiVersion))
		defer os.RemoveAll(tempDir1)

		tempDir2 := createTempDir("secondDir")
		writeToFile(filepath.Join(tempDir2, "prop2.xlvals"), "replaceme=success2\noverrideme=notoverriden")
		writeToFile(filepath.Join(tempDir2, "prop3.xlvals"), "overrideme=OVERRIDDEN")
		yaml2 := writeToTempFile(tempDir2, "yaml2", fmt.Sprintf(`
apiVersion: %s
kind: Deployment
spec:
  package: !value overrideme
  environment: Environments/AWS Environment
  undeployDependencies: true
  orchestrators:
    - parallel-by-deployed
`, xl.XldApiVersion))
		defer os.RemoveAll(tempDir2)

		v := viper.GetViper()

		infra := CreateTestInfra(v)
		defer infra.shutdown()

		DoApply([]string{yaml.Name(), yaml2.Name()})

		assert.Len(t, infra.documents, 3)
		doc1 := infra.spec(0)[0]
		assert.Equal(t, doc1["package"], "Applications/PetPortal/1.0")
		assert.Equal(t, doc1["environment"], "Environments/env1")

		doc2 := infra.spec(1)[0]
		assert.Equal(t, doc2["package"], "Applications/PetPortal/2.0")
		assert.Equal(t, doc2["environment"], "Environments/env2")

		doc3 := infra.spec(2)[0]
		assert.Equal(t, doc3["package"], "OVERRIDDEN")
		assert.Equal(t, doc3["environment"], "Environments/AWS Environment")
	})
}
