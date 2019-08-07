package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataMapTable(t *testing.T) {
	t.Run("should print valid data table (left aligned)", func(t *testing.T) {
		data := map[string]interface{}{"test": "*****", "userName": "testing", "confirm": true}
		expected :=
			` -------------------------------- ----------------------------------------------------
| LABEL                          | VALUE                                              |
 -------------------------------- ----------------------------------------------------
| confirm                        | true                                               |
| test                           | *****                                              |
| userName                       | testing                                            |
 -------------------------------- ----------------------------------------------------
`
		assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 50, "", 1, false))
	})

	t.Run("should print valid data table with new lines (left aligned)", func(t *testing.T) {
		data := map[string]interface{}{"test": "*****", "userName": "testing\n123", "confirm": true}
		expected :=
			` -------------------------------- ----------------------------------------------------
| LABEL                          | VALUE                                              |
 -------------------------------- ----------------------------------------------------
| confirm                        | true                                               |
| test                           | *****                                              |
| userName                       | testing\n123                                       |
 -------------------------------- ----------------------------------------------------
`
		assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 50, "", 1, false))
	})

	t.Run("should print valid data table (right aligned)", func(t *testing.T) {
		data := map[string]interface{}{"test": "*****", "userName": "testing", "confirm": true}
		expected :=
			` -------------------------------- ----------------------------------------------------
|                          LABEL |                                              VALUE |
 -------------------------------- ----------------------------------------------------
|                        confirm |                                               true |
|                           test |                                              ***** |
|                       userName |                                            testing |
 -------------------------------- ----------------------------------------------------
`
		assert.Equal(t, expected, DataMapTable(&data, TableAlignRight, 30, 50, "", 1, false))
	})

	t.Run("should print valid data table with small size (left aligned) and with padding 0", func(t *testing.T) {
		data := map[string]interface{}{"test": "*****", "userName": "verylongtextfortesting", "confirm": true}
		expected :=
			` ------------------------------ ----------
|LABEL                         |VALUE     |
 ------------------------------ ----------
|confirm                       |true      |
|test                          |*****     |
|userName                      |verylong..|
 ------------------------------ ----------
`
		assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 10, "", 0, false))
	})

	t.Run("should print valid data table with small size (left aligned) and with padding 1", func(t *testing.T) {
		data := map[string]interface{}{"test": "*****", "userName": "verylongtextfortesting", "confirm": true}
		expected :=
			` -------------------------------- ------------
| LABEL                          | VALUE      |
 -------------------------------- ------------
| confirm                        | true       |
| test                           | *****      |
| userName                       | verylong.. |
 -------------------------------- ------------
`
		assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 10, "", 1, false))
	})

	t.Run("should print valid data table with small size (left aligned) and with padding 3", func(t *testing.T) {
		data := map[string]interface{}{"test": "*****", "userName": "verylongtextfortesting", "confirm": true}
		expected :=
			` ------------------------------------ ----------------
|   LABEL                            |   VALUE        |
 ------------------------------------ ----------------
|   confirm                          |   true         |
|   test                             |   *****        |
|   userName                         |   verylong..   |
 ------------------------------------ ----------------
`
		assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 10, "", 3, false))
	})

	t.Run("should print valid data table and remove empty values (left aligned)", func(t *testing.T) {
		data := map[string]interface{}{"test": "*****", "userName": "", "confirm": true}
		expected :=
			` -------------------------------- ----------------------------------------------------
| LABEL                          | VALUE                                              |
 -------------------------------- ----------------------------------------------------
| confirm                        | true                                               |
| test                           | *****                                              |
 -------------------------------- ----------------------------------------------------
`
		assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 50, "", 1, true))
	})

	t.Run("should print valid data table and remove none values (left aligned)", func(t *testing.T) {
		data := map[string]interface{}{"test": "*****", "userName": "none", "confirm": true}
		expected :=
			` -------------------------------- ----------------------------------------------------
| LABEL                          | VALUE                                              |
 -------------------------------- ----------------------------------------------------
| confirm                        | true                                               |
| test                           | *****                                              |
 -------------------------------- ----------------------------------------------------
`
		assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 50, "", 1, true))
	})

    t.Run("should print valid data table and remove space values (left aligned)", func(t *testing.T) {
        data := map[string]interface{}{"test": "*****", "userName": " ", "confirm": true}
        expected :=
            ` -------------------------------- ----------------------------------------------------
| LABEL                          | VALUE                                              |
 -------------------------------- ----------------------------------------------------
| confirm                        | true                                               |
| test                           | *****                                              |
 -------------------------------- ----------------------------------------------------
`
        assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 50, "", 1, true))
    })

    t.Run("should print valid data table and remove None values (left aligned)", func(t *testing.T) {
        data := map[string]interface{}{"test": "*****", "userName": "None", "confirm": true}
        expected :=
            ` -------------------------------- ----------------------------------------------------
| LABEL                          | VALUE                                              |
 -------------------------------- ----------------------------------------------------
| confirm                        | true                                               |
| test                           | *****                                              |
 -------------------------------- ----------------------------------------------------
`
        assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 50, "", 1, true))
    })
}
