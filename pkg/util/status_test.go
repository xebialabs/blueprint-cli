package util

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestDataMapTable(t *testing.T) {
    t.Run("should print valid data table (left aligned)", func(t *testing.T) {
        data := map[string]interface{} {"test": "*****", "userName": "testing", "confirm": true}
        expected := ` ______________________________ __________________________________________________
|KEY                           |VALUE                                             |
 ------------------------------ --------------------------------------------------
|confirm                       |true                                              |
|test                          |*****                                             |
|userName                      |testing                                           |
 ------------------------------ --------------------------------------------------
`
        assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 50, ""))
    })

    t.Run("should print valid data table (right aligned)", func(t *testing.T) {
        data := map[string]interface{} {"test": "*****", "userName": "testing", "confirm": true}
        expected := ` ______________________________ __________________________________________________
|                           KEY|                                             VALUE|
 ------------------------------ --------------------------------------------------
|                       confirm|                                              true|
|                          test|                                             *****|
|                      userName|                                           testing|
 ------------------------------ --------------------------------------------------
`
        assert.Equal(t, expected, DataMapTable(&data, TableAlignRight, 30, 50, ""))
    })

    t.Run("should print valid data table with small size (left aligned)", func(t *testing.T) {
        data := map[string]interface{} {"test": "*****", "userName": "verylongtextfortesting", "confirm": true}
        expected := ` ______________________________ __________
|KEY                           |VALUE     |
 ------------------------------ ----------
|confirm                       |true      |
|test                          |*****     |
|userName                      |verylong..|
 ------------------------------ ----------
`
        assert.Equal(t, expected, DataMapTable(&data, TableAlignLeft, 30, 10, ""))
    })
}
