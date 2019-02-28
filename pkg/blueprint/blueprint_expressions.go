package blueprint

import (
	"math"

	"github.com/Knetic/govaluate"
	"github.com/xebialabs/xl-cli/pkg/util"
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
}

// ProcessCustomExpression evaluates the expressions passed in the blueprint.yaml file using https://github.com/Knetic/govaluate
// {parameters} are the result of the spec -> parameters defined in the blueprint yaml. Parameters needs to be defined before use.
func ProcessCustomExpression(exStr string, parameters map[string]interface{}) (interface{}, error) {
	util.Verbose("[expression] Evaluating expression [%s]\n", exStr)

	expression, err := govaluate.NewEvaluableExpressionWithFunctions(exStr, functions)
	if err != nil {
		return nil, err
	}

	return expression.Evaluate(parameters)
}
