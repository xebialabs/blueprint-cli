package blueprint

import (
    "fmt"
    "github.com/Knetic/govaluate"
    "github.com/xebialabs/xl-cli/pkg/util"
    "math"
    "net/url"
    "os/user"
    "regexp"
    "strconv"
)

var functions = map[string]govaluate.ExpressionFunction{
	"strlen": func(args ...interface{}) (interface{}, error) {
		length := len(args[0].(string))
		return (float64)(length), nil
	},
	"max": func(args ...interface{}) (interface{}, error) {
		max := math.Max(args[0].(float64), args[1].(float64))
		return max, nil
	},
	"min": func(args ...interface{}) (interface{}, error) {
		min := math.Min(args[0].(float64), args[1].(float64))
		return min, nil
	},
	"ceil": func(args ...interface{}) (interface{}, error) {
		ceil := math.Ceil(args[0].(float64))
		return ceil, nil
	},
	"floor": func(args ...interface{}) (interface{}, error) {
		floor := math.Floor(args[0].(float64))
		return floor, nil
	},
	"round": func(args ...interface{}) (interface{}, error) {
		round := math.Round(args[0].(float64))
		return round, nil
	},
	"randPassword": func(args ...interface{}) (interface{}, error) {
		pass := util.GeneratePassword(16)
		return pass, nil
	},
	"string": func(args ...interface{}) (interface{}, error) {
		return fmt.Sprintf("%v", args[0]), nil
	},
	"regexMatch": func(args ...interface{}) (interface{}, error) {
	    if len(args) != 2 {
	        return nil, fmt.Errorf("invalid number of arguments for regex fn, expecting 2 got %d", len(args))
        }
        pattern := args[0].(string)
        value := fmt.Sprintf("%v", args[1])
        match, err := regexp.MatchString("^"+pattern+"$", value)
        if err != nil {
            return false, err
        }
        if !match {
            return false, nil
        }
	    return true, nil
    },
    "isFile": func(args ...interface{}) (interface{}, error) {
        currentUser, err := user.Current()
        if err != nil {
            return nil, fmt.Errorf("cannot get current user: %s", err.Error())
        }
        filePath := util.ExpandHomeDirIfNeeded(args[0].(string), currentUser)
        if util.PathExists(filePath, false) {
            return true, nil
        }
        return false, nil
    },
    "isDir": func(args ...interface{}) (interface{}, error) {
        currentUser, err := user.Current()
        if err != nil {
            return nil, fmt.Errorf("cannot get current user: %s", err.Error())
        }
        dirPath := util.ExpandHomeDirIfNeeded(args[0].(string), currentUser)
        if util.PathExists(dirPath, true) {
            return true, nil
        }
        return false, nil
    },
    "isValidUrl": func(args ...interface{}) (interface{}, error) {
        _, err := url.ParseRequestURI(args[0].(string))
        if err != nil {
            return false, nil
        }
        return true, nil
    },
}

// ProcessCustomExpression evaluates the expressions passed in the blueprint.yaml file using https://github.com/Knetic/govaluate
// {parameters} are the result of the spec -> parameters defined in the blueprint yaml. Parameters needs to be defined before use.
func ProcessCustomExpression(exStr string, parameters map[string]interface{}, currentKey string, currentVal interface{}) (interface{}, error) {
	util.Verbose("[expression] Evaluating expression [%s]\n", exStr)

	expression, err := govaluate.NewEvaluableExpressionWithFunctions(exStr, functions)
	if err != nil {
		return nil, err
	}

	// add this value to the map of parameters for expression
	expressionParams := fixValueTypes(parameters)
	if currentKey != "" {
        expressionParams[currentKey] = currentVal
    }
	return expression.Evaluate(expressionParams)
}

func fixValueTypes(parameters map[string]interface{}) map[string]interface{} {
	newParams := make(map[string]interface{})
	for k, v := range parameters {
		switch vStr := v.(type) {
		case string:
			if val, err := strconv.ParseFloat(vStr, 64); err == nil {
				newParams[k] = val
			} else if val, err := strconv.ParseBool(vStr); err == nil {
				newParams[k] = val
			} else {
				newParams[k] = vStr
			}
		default:
			newParams[k] = v
		}
	}
	return newParams
}
