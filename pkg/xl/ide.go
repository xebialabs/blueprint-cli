package xl

import (
	"fmt"
	"io/ioutil"
	"os"
)

func (c *Context) GenerateSchema(schemaFilename string, generateXld bool, generateXlr bool, override bool) error {
	if generateXld {
		if _, err := os.Stat(schemaFilename); !override && !os.IsNotExist(err) {
			return fmt.Errorf("file `%s` already exists", schemaFilename)
		}
		server := c.XLDeploy
		Info("Downloading XL Deploy IDE Schema...\n")
		bytes, e := server.GetSchema()
		if e != nil {
			return e
		}
		e2 := ioutil.WriteFile(schemaFilename, bytes, 0644)
		if e2 != nil {
			return e2
		}
	}
	if generateXlr {
		// Todo implement support for XLR and merge schemas
		return fmt.Errorf("XL Release not yet supported")
	}

	return nil
}
