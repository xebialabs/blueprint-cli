package xl

import (
	"strings"
	"fmt"
)

func Substitute(in string, values map[string]string) (res string, err error) {
	var result strings.Builder
	var capture strings.Builder
	var capturing = false
	for _, rune := range in {
		runeS := string(rune)
		if runeS == "%" && capturing == false {
			capturing = true
		} else if runeS == "%" {
			if capture.String() == "" {
				result.WriteString("%")
			} else {
				if val, ok := values[capture.String()]; ok {
					Verbose("\tSubstituting value for [%s]\n", capture.String())
					result.WriteString(val)
				} else {
					return "", fmt.Errorf("unknown value: `%s`", capture.String())
				}
			}
			capture.Reset()
			capturing = false
		} else if capturing == true {
			capture.WriteString(runeS)
		} else {
			result.WriteString(runeS)
		}
	}
	if capturing {
		return "", fmt.Errorf("invalid format string: `%s`", in)
	} else {
		return result.String(), nil
	}
}
