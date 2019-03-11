package xl

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"github.com/xebialabs/xl-cli/pkg/util"
    "os"
    "fmt"
)

type reference struct {
	Ref string `json:"$ref"`
}

type jsonSchema struct {
	Definitions map[string]interface{} `json:"definitions"`
	SchemaType  string                 `json:"type"`
	Schema      string                 `json:"$schema"`
	Description string                 `json:"description"`
	OneOf       []reference            `json:"oneOf"`
}

func referenceInit(xldSchema []byte, xlrSchema []byte) []reference {
	if xldSchema != nil && xlrSchema != nil {
		references := make([]reference, 2)
		references[0] = reference{Ref: "#/definitions/xld"}
		references[1] = reference{Ref: "#/definitions/xlr"}

		return references
	} else {
		references := make([]reference, 1)

		if xldSchema != nil {
			references[0] = reference{Ref: "#/definitions/xld"}
		} else {
			references[0] = reference{Ref: "#/definitions/xlr"}
		}

		return references
	}
}

func processSchemas(xldSchema []byte, xlrSchema []byte, schemaFilename string) error {
	var xldSchemaJson interface{}
	var xlrSchemaJson interface{}
	definitions := make(map[string]interface{})

	if xldSchema != nil {
		err := json.Unmarshal(xldSchema, &xldSchemaJson)

		if err != nil {
			return err
		}
		definitions["xld"] = xldSchemaJson.(map[string]interface{})["definitions"].(map[string]interface{})["xld"]
	}

	if xlrSchema != nil {
		err := json.Unmarshal(xlrSchema, &xlrSchemaJson)

		if err != nil {
			return err
		}
		definitions["xlr"] = xlrSchemaJson.(map[string]interface{})["definitions"].(map[string]interface{})["xlr"]
	}

	var references = referenceInit(xldSchema, xlrSchema)

	rootSchema := jsonSchema{
		Definitions: definitions,
		SchemaType:  "object",
		Schema:      "http://json-schema.org/schema#",
		Description: "A description of objects in the XL Products",
		OneOf:       references,
	}

	bodyBytes := new(bytes.Buffer)
	e := json.NewEncoder(bodyBytes).Encode(rootSchema)
	if e != nil {
		return e
	}

	e2 := ioutil.WriteFile(schemaFilename, bodyBytes.Bytes(), 0644)
	if e2 != nil {
		return e2
	}

	return nil
}

func (c *Context) GenerateSchema(schemaFilename string, generateXld bool, generateXlr bool, override bool) error {
	var xldSchema []byte = nil
	var xlrSchema []byte = nil

	if generateXlr || generateXld {
        if _, err := os.Stat(schemaFilename); !override && !os.IsNotExist(err) {
            return fmt.Errorf("file `%s` already exists", schemaFilename)
        }
    }

	if generateXld {
		server := c.XLDeploy
		util.Info("Downloading XL Deploy IDE Schema\n")
		schemaBytes, e := server.GetSchema()
		if e != nil {
			return e
		}
		xldSchema = schemaBytes
	} else {
        util.Info("Skipping XL Deploy IDE Schema\n")
    }
	if generateXlr {
        server := c.XLRelease
        util.Info("Downloading XL Release IDE Schema\n")
        schemaBytes, e := server.GetSchema()
        if e != nil {
            return e
        }
        xlrSchema = schemaBytes
	} else {
        util.Info("Skipping XL Release IDE Schema\n")
    }

    util.Info("Processing IDE Schemas and writing to '%s'\n", schemaFilename)
	return processSchemas(xldSchema, xlrSchema, schemaFilename)
}
