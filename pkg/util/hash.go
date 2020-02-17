package util

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	funk "github.com/thoas/go-funk"
)

func Md5HashFromMap(params map[string]interface{}) (string, error) {
	valueBytes, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	md5Sum := md5.Sum(valueBytes)
	return fmt.Sprintf("%x", md5Sum), nil
}

func Md5HashFromFilteredMap(params map[string]interface{}, filters []interface{}, invert bool) (string, error) {
	if len(filters) > 0 {
		filteredParams := make(map[string]interface{})
		for k, v := range params {
			if invert {
				// add items that are not filtered
				if !funk.Contains(filters, k) {
					filteredParams[k] = v
				}
			} else {
				// add items that are filtered
				if funk.Contains(filters, k) {
					filteredParams[k] = v
				}
			}
		}
		return Md5HashFromMap(filteredParams)
	}
	return Md5HashFromMap(params)
}
