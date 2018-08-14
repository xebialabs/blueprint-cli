package lib

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"strings"
)

func TestYamlParser(t *testing.T) {
	t.Run("should parse Yaml document", func(t *testing.T) {
		yaml := `
apiVersion: xl-deploy/v1
kind: Applications
metadata:
  Applications-home: Applications/Cloud
spec:
- name: Applications/AWS
  type: core.Directory
  children:
    - name: rest-o-rant-ecs-service
      type: udm.Application
      children:
      - name: 1.0
        type: udm.DeploymentPackage
`

		reader := strings.NewReader(yaml)
		docreader := NewDocumentReader(reader)
		doc, err := docreader.ReadNextYamlDocument()

		assert.Nil(t, err)
		assert.NotNil(t, doc)

		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "xl-deploy/v1", doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)
		assert.NotNil(t, doc.Metadata)
		assert.Equal(t, "Applications/Cloud", doc.Metadata["Applications-home"])
		assert.NotNil(t, doc.Spec)
		assert.Equal(t, "Applications/AWS", doc.Spec[0]["name"])
		children := doc.Spec[0]["children"].([]interface{})
		firstChild := children[0].(map[interface{}]interface{})
		assert.Equal(t, "rest-o-rant-ecs-service", firstChild["name"])
		children2 := firstChild["children"].([]interface{})
		firstChild2 := children2[0].(map[interface{}]interface{})
		assert.Equal(t, "udm.DeploymentPackage", firstChild2["type"])
	} )

	t.Run("should parse Yaml file with multiple documents", func(t *testing.T) {
		yaml := `
apiVersion: xl-deploy/v1
kind: Applications
spec:
- name: Applications/AWS1

---

apiVersion: xl-release/v1
kind: Template
spec:
- name: Template1
`

		reader := strings.NewReader(yaml)
		docreader := NewDocumentReader(reader)

		doc, err := docreader.ReadNextYamlDocument()
		assert.Nil(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "xl-deploy/v1", doc.ApiVersion)
		assert.Equal(t, "Applications", doc.Kind)
		assert.NotNil(t, doc.Spec)
		assert.Equal(t, "Applications/AWS1", doc.Spec[0]["name"])

		doc2, err := docreader.ReadNextYamlDocument()
		assert.Nil(t, err)
		assert.NotNil(t, doc2)
		assert.Equal(t, "xl-release/v1", doc2.ApiVersion)
		assert.Equal(t, "Template", doc2.Kind)
		assert.NotNil(t, doc2.Spec)
		assert.Equal(t, "Template1", doc2.Spec[0]["name"])
	} )
}


