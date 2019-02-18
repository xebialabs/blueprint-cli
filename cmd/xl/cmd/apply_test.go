package cmd

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"os"
	"path/filepath"
	"testing"
)

func TestApply(t *testing.T) {

	util.IsVerbose = true

	t.Run("should apply multiple yaml files in right order with value replacement to both xlr and xld", func(t *testing.T) {
		tempDir1 := createTempDir("firstDir")
		writeToFile(filepath.Join(tempDir1, "prop1.xlvals"), "replaceme=success1")
		yaml := writeToTempFile(tempDir1, "yaml1", fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template1
- replaceTest: !value replaceme

---

apiVersion: %s
kind: Applications
spec:
- name: App1
`, xl.XlrApiVersion, xl.XldApiVersion))
		defer os.RemoveAll(tempDir1)

		tempDir2 := createTempDir("secondDir")
		writeToFile(filepath.Join(tempDir2, "prop2.xlvals"), "replaceme=success2\noverrideme=notoverriden")
		writeToFile(filepath.Join(tempDir2, "prop3.xlvals"), "overrideme=OVERRIDDEN")
		yaml2 := writeToTempFile(tempDir2, "yaml2", fmt.Sprintf(`
apiVersion: %s
kind: Template
spec:
- name: Template2
- replaceTest: !value replaceme
- overrideTest: !value overrideme
---

apiVersion: %s
kind: Applications
spec:
- name: App2
`, xl.XlrApiVersion, xl.XldApiVersion))
		defer os.RemoveAll(tempDir2)

		v := viper.GetViper()
		v.Set("xl-deploy.applications-home", "Applications/XL")
		v.Set("xl-release.home", "XL")

		infra := CreateTestInfra(v)
		defer infra.shutdown()

		DoApply([]string{yaml.Name(), yaml2.Name()})

		assert.Equal(t, infra.spec(0)[0]["name"], "Template1")
		assert.Equal(t, infra.spec(0)[1]["replaceTest"], "success1")
		assert.Equal(t, infra.metadata(0)["home"], "XL")

		assert.Equal(t, infra.spec(1)[0]["name"], "App1")
		assert.Equal(t, infra.metadata(1)["Applications-home"], "Applications/XL")

		assert.Equal(t, infra.spec(2)[0]["name"], "Template2")
		assert.Equal(t, infra.spec(2)[1]["replaceTest"], "success2")
		assert.Equal(t, infra.spec(2)[2]["overrideTest"], "OVERRIDDEN")
		assert.Equal(t, infra.metadata(2)["home"], "XL")

		assert.Equal(t, infra.spec(3)[0]["name"], "App2")
		assert.Equal(t, infra.metadata(3)["Applications-home"], "Applications/XL")
	})

	t.Run("should take imports into account", func(t *testing.T) {
		tempDir := createTempDir("imports")
		provisionFile := writeToTempFile(tempDir, "provision.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Applications
spec:
- name: App1
`, xl.XldApiVersion))

		deployFile := writeToTempFile(tempDir, "deploy.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Deployment
metadata:
  imports:
    - %s
spec:
  package: Applications/PetPortal/1.0
  environment: Environments/AWS Environment
  undeployDependencies: true
  orchestrators:
  - parallel-by-deployment-group
  - sequential-by-container
`, xl.XldApiVersion, filepath.Base(provisionFile.Name())))
		defer os.RemoveAll(tempDir)

		v := viper.GetViper()
		infra := CreateTestInfra(v)
		defer infra.shutdown()

		DoApply([]string{deployFile.Name()})

		assert.Equal(t, len(infra.documents), 2)
		assert.Equal(t, infra.doc(0).Kind, "Applications")
		assert.Equal(t, infra.doc(1).Kind, "Deployment")
		assert.Nil(t, infra.metadata(1)["imports"])
	})

	t.Run("should support 'imports' file together with imports inside of imported files", func(t *testing.T) {
		tempDir := createTempDir("imports2")
		provisionFile := writeToTempFile(tempDir, "provision.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Applications
spec:
- name: App1
`, xl.XldApiVersion))

		deployFile := writeToTempFile(tempDir, "deploy.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Deployment
metadata:
  imports:
    - %s
spec:
  package: Applications/PetPortal/1.0
  environment: Environments/AWS Environment
  undeployDependencies: true
  orchestrators:
  - parallel-by-deployment-group
  - sequential-by-container
`, xl.XldApiVersion, filepath.Base(provisionFile.Name())))

		importsFile := writeToTempFile(tempDir, "imports.yaml", fmt.Sprintf(`
apiVersion: %s
kind: Import
metadata:
  imports:
    - %s
`, models.YamlFormatVersion, filepath.Base(deployFile.Name())))
		defer os.RemoveAll(tempDir)

		v := viper.GetViper()
		infra := CreateTestInfra(v)
		defer infra.shutdown()

		DoApply([]string{importsFile.Name()})

		assert.Equal(t, len(infra.documents), 2)
		assert.Equal(t, infra.doc(0).Kind, "Applications")
		assert.Equal(t, infra.doc(1).Kind, "Deployment")
		assert.Nil(t, infra.metadata(1)["imports"])
	})
}
