package k8s

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/xebialabs/xl-cli/pkg/util"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	requestPresignParam    = 60
	presignedURLExpiration = 15 * time.Minute
	v1Prefix               = "k8s-aws-v1."
	maxTokenLenBytes       = 1024 * 4
	clusterIDHeader        = "x-k8s-aws-id"
	dateHeaderFormat       = "20060102T150405Z"
	hostRegexp             = `^sts(\.[a-z1-9\-]+)?\.amazonaws\.com(\.cn)?$`
)

var parameterWhitelist = map[string]bool{
	"action":               true,
	"version":              true,
	"x-amz-algorithm":      true,
	"x-amz-credential":     true,
	"x-amz-date":           true,
	"x-amz-expires":        true,
	"x-amz-security-token": true,
	"x-amz-signature":      true,
	"x-amz-signedheaders":  true,
}

// Token is generated and used by Kubernetes client-go to authenticate with a Kubernetes cluster.
type Token struct {
	Token      string
	Expiration time.Time
}

type getCallerIdentityWrapper struct {
	GetCallerIdentityResponse struct {
		GetCallerIdentityResult struct {
			Account string `json:"Account"`
			Arn     string `json:"Arn"`
			UserID  string `json:"UserId"`
		} `json:"GetCallerIdentityResult"`
		ResponseMetadata struct {
			RequestID string `json:"RequestId"`
		} `json:"ResponseMetadata"`
	} `json:"GetCallerIdentityResponse"`
}

func connectToEKS(answerMap map[string]string) (*restclient.Config, error) {
	fmt.Println("Connecting to EKS")
	clusterID := getClusterIDFromClusterName(answerMap)

	var sess *session.Session
	var err error

	if util.MapContainsKeyWithVal(answerMap, "AWSAccessKey") {
		var region string

		if util.MapContainsKeyWithVal(answerMap, "AWSRegion") {
			region, err = GetRequiredPropertyFromMap("AWSRegion", answerMap)
			if err != nil {
				return nil, err
			}
		} else {
			region = "eu-west-1"
		}
		AWSAccessKey, err := GetRequiredPropertyFromMap("AWSAccessKey", answerMap)
		if err != nil {
			return nil, err
		}
		AWSAccessSecret, err := GetRequiredPropertyFromMap("AWSAccessSecret", answerMap)
		if err != nil {
			return nil, err
		}
		sess, err = session.NewSession(&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(AWSAccessKey, AWSAccessSecret, ""),
		})
	} else {
		sess, err = session.NewSessionWithOptions(session.Options{
			AssumeRoleTokenProvider: stdinStderrTokenProvider,
			SharedConfigState:       session.SharedConfigEnable,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("could not create session: %v", err)
	}

	stsAPI := sts.New(sess)

	request, _ := stsAPI.GetCallerIdentityRequest(&sts.GetCallerIdentityInput{})
	request.HTTPRequest.Header.Add(clusterIDHeader, clusterID)

	presignedURLString, err := request.Presign(requestPresignParam)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse the request %s", err)
	}

	tokenExpiration := time.Now().Local().Add(presignedURLExpiration - 1*time.Minute)
	t := Token{v1Prefix + base64.RawURLEncoding.EncodeToString([]byte(presignedURLString)), tokenExpiration}

	url, err := GetRequiredPropertyFromMap("K8sApiServerURL", answerMap)
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.BuildConfigFromFlags(url, "")
	if err != nil {
		return nil, err
	}
	config.TLSClientConfig.Insecure = true
	config.BearerToken = t.Token

	return config, nil
}

// GetK8sConfiguration gets the Kubernetes connection configuration
func GetK8sConfiguration(answerMap map[string]string) (*restclient.Config, error) {
	if answerMap["K8sSetup"] == "AwsEKS" {
		return connectToEKS(answerMap)
	}

	return connectToK8s(answerMap)
}

func connectToK8s(answerMap map[string]string) (*restclient.Config, error) {
	url, err := GetRequiredPropertyFromMap("K8sApiServerURL", answerMap)
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.BuildConfigFromFlags(url, "")
	if err != nil {
		return nil, err
	}

	if util.MapContainsKeyWithVal(answerMap, "K8sToken") {
		config.BearerToken, err = GetRequiredPropertyFromMap("K8sToken", answerMap)
		if err != nil {
			return nil, err
		}
	} else {

		if util.MapContainsKeyWithVal(answerMap, "K8sClientCert") {
			data, err := GetRequiredPropertyFromMap("K8sClientCert", answerMap)
			if err != nil {
				return nil, err
			}
			config.CertData, err = DecodeBase64(data)
			if err != nil {
				return nil, err
			}
		} else {
			config.CertFile, err = GetRequiredPropertyFromMap("K8sClientCertFile", answerMap)
			if err != nil {
				return nil, err
			}
		}

		if util.MapContainsKeyWithVal(answerMap, "K8sClientKey") {
			data, err := GetRequiredPropertyFromMap("K8sClientKey", answerMap)
			if err != nil {
				return nil, err
			}
			config.KeyData, err = DecodeBase64(data)
			if err != nil {
				return nil, err
			}
		} else {
			config.KeyFile, err = GetRequiredPropertyFromMap("K8sClientKeyFile", answerMap)
			if err != nil {
				return nil, err
			}
		}
	}
	// TODO check this connection param
	config.TLSClientConfig.Insecure = true

	return config, nil
}

func stdinStderrTokenProvider() (string, error) {
	var v string
	fmt.Fprint(os.Stderr, "Assume Role MFA token code: ")
	_, err := fmt.Scanln(&v)
	return v, err
}

func DecodeBase64(data string) ([]byte, error) {
	decode, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("error decoding the certificates %s", err)
	}
	return decode, nil
}

func GetRequiredPropertyFromMap(propertyName string, answerMap map[string]string) (string, error) {
	value := answerMap[propertyName]
	if value == "" {
		return "", fmt.Errorf("the property %s is required to connect with Kubernetes", propertyName)
	}
	return value, nil
}

func getClusterIDFromClusterName(answerMap map[string]string) string {
	clusterName := answerMap["EksClusterName"]

	if clusterName != "" {
		return clusterName
	}

	return "xl-up-master"
}
