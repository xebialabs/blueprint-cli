package blueprint

import (
	"fmt"
	"math"
	"net/url"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thoas/go-funk"

	"github.com/Knetic/govaluate"
	"github.com/dlclark/regexp2"
	"github.com/xebialabs/blueprint-cli/pkg/cloud/aws"
	"github.com/xebialabs/blueprint-cli/pkg/cloud/k8s"
	"github.com/xebialabs/blueprint-cli/pkg/osHelper"
	"github.com/xebialabs/blueprint-cli/pkg/util"
	versionHelper "github.com/xebialabs/blueprint-cli/pkg/version"
)

func regexMatch(pattern, value string) (bool, error) {
	re, err := regexp2.Compile(pattern, 0)
	if err != nil {
		return false, fmt.Errorf("invalid pattern in regex expression, %s", err.Error())
	}
	// setting a 5 second timeout to avoid hanging on complex regex
	re.MatchTimeout = time.Second * 5
	match, err := re.MatchString(value)

	if err != nil {
		return false, fmt.Errorf("error while matching regex expression %s, %s", pattern, err.Error())
	}

	if !match {
		return false, nil
	}
	return true, nil
}

func getExpressionFunctions(params map[string]interface{}) map[string]govaluate.ExpressionFunction {
	return map[string]govaluate.ExpressionFunction{
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
			pattern := fmt.Sprintf("^%s$", args[0].(string))
			value := fmt.Sprintf("%v", args[1])
			return regexMatch(pattern, value)
		},
		"isValidAbsPath": func(args ...interface{}) (interface{}, error) {
			path := args[0].(string)
			windPathRegex := `[a-zA-Z]:\\(((?![<>:"/\\|?*]).)+((?<![ .])\\)?)*` // windows absolute path with space
			unixPathRegex := `(.?\/[\w^ -]+)*\/?([\w-])+[.]?[^.\s]*`            // unix absolute path with space
			if len(args) == 2 && fmt.Sprintf("%v", args[1]) == "true" {
				// update regex to validate path without spaces in it
				windPathRegex = `[a-zA-Z]:\\(((?![<>:"/\\|?*])[\S])+((?<![ .])\\)?)*` // windows absolute path with out space
				unixPathRegex = `(.?\/[\w^-]+)*\/?([\w-])+[.]?[^.\s]*`                // unix absolute path with out space
			}
			pattern := fmt.Sprintf("^(%s|%s)$", windPathRegex, unixPathRegex)
			return regexMatch(pattern, path)
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
			// - 0: Config attribute name [ClusterServer, ClusterInsecureSkipTLSVerify, ContextCluster, ContextNamespace, ContextUser, UserClientCertificateData, UserClientKeyData, IsAvailable, IsConfigAvailable]
			// - 1: Context name [optional]
			attr := fmt.Sprintf("%v", args[0])

			if !funk.Contains([]string{"ClusterServer", "ClusterInsecureSkipTLSVerify", "ContextCluster", "ContextNamespace", "ContextUser", "UserClientCertificateData", "UserClientKeyData", "IsAvailable", "IsConfigAvailable"}, attr) {
				return nil, fmt.Errorf("attribute '%s' is not valid for expression function 'k8sConfig'", attr)
			}

			kubeConfigPresent := k8s.IsKubeConfigFilePresent()
			if attr == "IsConfigAvailable" {
				return kubeConfigPresent, nil
			}

			if !kubeConfigPresent {
				if attr == "IsAvailable" {
					return false, nil
				} else {
					return "", nil
				}
			}

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
			return k8sConfig.GetConfigField(attr, true), nil
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

			if !funk.Contains([]string{"_defaultapiserverurl", "_operatingsystem", "getcertfilelocation", "getkeyfilelocation"}, module) {
				return nil, fmt.Errorf("attribute '%s' is not valid for expression function 'os'", module)
			}
			return osHelper.GetPropertyByName(module)
		},

		// xl up helper functions
		"version": func(args ...interface{}) (interface{}, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("invalid number of arguments for  expression function 'xlUp', expecting 1 (module name) got %d", len(args))
			}
			/* Available modules:
			   - _showapplicableversions
			   - checkversion
			   - getversionfromtag
			*/

			var params []string
			module := fmt.Sprintf("%v", args[0])

			if !funk.Contains([]string{"_showapplicableversions", "checkversion", "getversionfromtag"}, module) {
				return nil, fmt.Errorf("attribute '%s' is not valid for expression function 'version'", module)
			}

			if len(args) == 2 && module == "_showapplicableversions" {
				params = append(params, fmt.Sprintf("%v", args[1]))
			}

			if len(args) == 3 && module == "checkversion" {
				params = append(params, fmt.Sprintf("%v", args[1]))
				params = append(params, fmt.Sprintf("%v", args[2]))
			}

			if len(args) == 2 && module == "getversionfromtag" {
				params = append(params, fmt.Sprintf("%v", args[1]))
			}

			result, err := versionHelper.GetPropertyByName(module, params...)
			if err != nil {
				return nil, fmt.Errorf("Error when executing expression function '%s', %s", module, err.Error())
			}
			return result, nil
		},

		// normalize windows paths to mountable docker paths
		"normalizePath": func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("invalid number of arguments for  expression function 'normalizePath', expecting 1 (module name) got %d", len(args))
			}

			path := fmt.Sprintf("%v", args[0])

			pathRe := regexp.MustCompile(`^([A-Za-z]):\\`)
			path = pathRe.ReplaceAllString(path, `/$1/`)
			path = strings.ReplaceAll(path, "\\", `/`)

			return path, nil
		},

		// create a unique hash of all answer values
		"md5HashOfAnswersIncluding": func(args ...interface{}) (interface{}, error) {
			return util.Md5HashFromFilteredMap(params, args, false)
		},
		// create a unique hash of all answer values
		"md5HashOfAnswersNotIncluding": func(args ...interface{}) (interface{}, error) {
			return util.Md5HashFromFilteredMap(params, args, true)
		},
	}
}

// ProcessCustomExpression evaluates the expressions passed in the blueprint.yaml file using https://github.com/Knetic/govaluate
// {parameters} are the result of the spec -> parameters defined in the blueprint yaml. Parameters needs to be defined before use.
func ProcessCustomExpression(exStr string, parameters map[string]interface{}) (interface{}, error) {
	util.Verbose("[expression] Evaluating expression [%s]\n", exStr)

	expressionParams := FixValueTypes(parameters)

	expression, err := govaluate.NewEvaluableExpressionWithFunctions(exStr, getExpressionFunctions(expressionParams))
	if err != nil {
		return nil, err
	}

	return expression.Evaluate(expressionParams)
}

func FixValueTypes(parameters map[string]interface{}) map[string]interface{} {
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
