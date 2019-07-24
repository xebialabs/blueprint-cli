package blueprint

import (
	"fmt"
	"math"
	"net/url"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/thoas/go-funk"

	"github.com/Knetic/govaluate"
	"github.com/dlclark/regexp2"
	"github.com/xebialabs/xl-cli/pkg/cloud/aws"
	"github.com/xebialabs/xl-cli/pkg/cloud/k8s"
	"github.com/xebialabs/xl-cli/pkg/osHelper"
	"github.com/xebialabs/xl-cli/pkg/util"
	upHelper "github.com/xebialabs/xl-cli/pkg/version"
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
	"regex": func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("invalid number of arguments for regex expression, expecting 2 got %d", len(args))
		}
		pattern := args[0].(string)
		re, err := regexp2.Compile(fmt.Sprintf("^%s$", pattern), 0)
		if err != nil {
			return false, fmt.Errorf("invalid pattern in regex expression, %s", err.Error())
		}
		// setting a 5 second timeout to avoid hanging on complex regex
		re.MatchTimeout = time.Second * 5
		value := fmt.Sprintf("%v", args[1])
		match, err := re.MatchString(value)

		if err != nil {
			return false, fmt.Errorf("error while matching regex expression %s, %s", pattern, err.Error())
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

	// aws helper functions
	"awsCredentials": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("invalid number of arguments for expression function 'awsCredentials', expecting 1 got %d", len(args))
		}

		// possible attributes: [IsAvailable, AccessKeyID, SecretAccessKey, ProviderName]
		attr := fmt.Sprintf("%v", args[0])
		if !funk.Contains([]string{"IsAvailable", "AccessKeyID", "SecretAccessKey", "ProviderName"}, attr) {
			return nil, fmt.Errorf("attribute '%s' is not valid for expression function 'awsCredentials'", attr)
		}

		creds, err := aws.GetAWSCredentialsFromSystem()
		if err != nil {
			if strings.Contains(err.Error(), "NoCredentialProviders: no valid providers in chain") {
				if attr == "IsAvailable" {
					return false, nil
				}
				return nil, nil
			}
			return nil, fmt.Errorf("Error when executing expression function 'awsCredentials', %s", err.Error())
		}

		if attr == "IsAvailable" {
			return creds.AccessKeyID != "", nil
		}
		return aws.GetAWSCredentialsField(&creds, attr), nil
	},
	"awsRegions": func(args ...interface{}) (interface{}, error) {
		if len(args) == 0 || len(args) > 2 {
			return nil, fmt.Errorf("invalid number of arguments for expression function 'awsRegions', expecting between 1 and 2 got %d", len(args))
		}

		// attributes:
		// - 0: AWS service name
		// - 1: Index of the result list [optional]
		serviceName := fmt.Sprintf("%v", args[0])
		i := -1
		var err error
		if len(args) == 2 {
			i, err = strconv.Atoi(fmt.Sprintf("%v", args[1]))
			if err != nil {
				return nil, fmt.Errorf("second argument for expression function 'awsRegions' should be a number")
			}
		}

		regions, err := aws.GetAvailableAWSRegionsForService(serviceName)
		if err != nil {
			return nil, fmt.Errorf("Error when executing expression function 'awsRegions', %s", err.Error())
		}

		if i >= len(regions) {
			return nil, fmt.Errorf("index %d doesn't exist in the result of expression function 'awsRegions'", i)
		}
		if i >= 0 {
			return regions[i], nil
		}
		return regions, nil
	},

	// k8s helper functions
	"k8sConfig": func(args ...interface{}) (interface{}, error) {
		if len(args) == 0 || len(args) > 2 {
			return nil, fmt.Errorf("invalid number of arguments for expression function 'k8sConfig', expecting between 1 and 2 got %d", len(args))
		}

		// attributes:
		// - 0: Config attribute name [ClusterServer, ClusterInsecureSkipTLSVerify, ContextCluster, ContextNamespace, ContextUser, UserClientCertificateData, UserClientKeyData, IsAvailable]
		// - 1: Context name [optional]
		attr := fmt.Sprintf("%v", args[0])
		contextName := ""
		if len(args) == 2 {
			contextName = fmt.Sprintf("%v", args[1])
		}
		k8sConfig, err := k8s.GetK8SConfigFromSystem(contextName)
		if err != nil {
			if strings.Contains(err.Error(), "Specified context was not found in the Kubernetes config file") && attr == "IsAvailable" {
				return false, nil
			}
			return nil, fmt.Errorf("Error when executing expression function 'k8sConfig', %s", err.Error())
		}

		if attr == "IsAvailable" {
			return k8sConfig.Cluster.Server != "", nil
		}
		return k8sConfig.GetConfigField(attr), nil
	},

	// os helper functions
	"os": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("invalid number of arguments for expression function 'os', expecting 1 (module name) got %d", len(args))
		}

		/* Available modules:
		   - _defaultapiserverurl
		   - _operatingsystem
		   - getcertfilelocation
		   - getkeyfilelocation
		*/
		module := fmt.Sprintf("%v", args[0])
		result, err := osHelper.GetPropertyByName(module)

		if err != nil {
			return nil, fmt.Errorf("Error when executing expression function '%s', %s", module, err.Error())
		}

		if module == "_defaultapiserverurl" {
			return result[0], nil
		}

		return result, err
	},

	// xl up helper functions
	"xlUp": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("invalid number of arguments for  expression function 'xlUp', expecting 1 (module name) got %d", len(args))
		}

		/* Available modules:
		   - _showapplicableversions
		*/
		module := fmt.Sprintf("%v", args[0])
		return upHelper.GetPropertyByName(module)
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

	expressionParams := fixValueTypes(parameters)
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
