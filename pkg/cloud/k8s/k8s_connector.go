package k8s

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

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

func connectToEKS(answerMap map[string]string) *restclient.Config {
    fmt.Println("Connecting to EKS")
	clusterID := getClusterIDFromClusterName(answerMap)

	sess, err := session.NewSessionWithOptions(session.Options{
		AssumeRoleTokenProvider: stdinStderrTokenProvider,
		SharedConfigState:       session.SharedConfigEnable,
	})

	if err != nil {
		util.Fatal("could not create session: %v", err)
	}

	stsAPI := sts.New(sess)

	request, _ := stsAPI.GetCallerIdentityRequest(&sts.GetCallerIdentityInput{})
	request.HTTPRequest.Header.Add(clusterIDHeader, clusterID)

	presignedURLString, err := request.Presign(requestPresignParam)
	if err != nil {
		util.Fatal("Cannot parse the request %s", err)
	}

	tokenExpiration := time.Now().Local().Add(presignedURLExpiration - 1*time.Minute)
	t := Token{v1Prefix + base64.RawURLEncoding.EncodeToString([]byte(presignedURLString)), tokenExpiration}

	url := GetRequiredPropertyFromMap("apiServerURL", answerMap)

	config, err := clientcmd.BuildConfigFromFlags(url, "")
	if err != nil {
		panic(err.Error())
	}

	config.TLSClientConfig.Insecure = true
	config.BearerToken = t.Token

	return config
}

// GetK8sConfiguration gets the Kubernetes connection configuration
func GetK8sConfiguration(answerMap map[string]string) *restclient.Config {
	if answerMap["IsEKS"] == "true" {
		return connectToEKS(answerMap)
	}

	return connectToK8s(answerMap)
}

func connectToK8s(answerMap map[string]string) *restclient.Config {
	url := GetRequiredPropertyFromMap("apiServerURL", answerMap)

	config, err := clientcmd.BuildConfigFromFlags(url, "")
	if err != nil {
		panic(err.Error())
	}

    if IsPropertyPresent("K8sToken", answerMap) {
        config.BearerToken = GetRequiredPropertyFromMap("K8sToken", answerMap)
    } else {

        if IsPropertyPresent("K8sClientCert", answerMap) {
            config.CertData = DecodeBase64(GetRequiredPropertyFromMap("K8sClientCert", answerMap))
        } else {
            config.CertFile = GetRequiredPropertyFromMap("K8sClientCertFile", answerMap)
        }

        if IsPropertyPresent("K8sClientKey", answerMap) {
            config.KeyData = DecodeBase64(GetRequiredPropertyFromMap("K8sClientKey", answerMap))
        } else {
            config.KeyFile = GetRequiredPropertyFromMap("K8sClientKeyFile", answerMap)
        }
    }
	// TODO check this connection param
	config.TLSClientConfig.Insecure = true

	return config
}

func stdinStderrTokenProvider() (string, error) {
	var v string
	fmt.Fprint(os.Stderr, "Assume Role MFA token code: ")
	_, err := fmt.Scanln(&v)
	return v, err
}

func DecodeBase64(data string) []byte {
	decode, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		util.Fatal("Error decoding the certificates")
	}
	return decode
}

func GetRequiredPropertyFromMap(propertyName string, answerMap map[string]string) string {
	value := answerMap[propertyName]
	if value == "" {
		util.Fatal("The property %s is required to connect with Kubernetes", propertyName)
	}
	return value
}

func IsPropertyPresent(propertyName string, answerMap map[string]string) bool {
	value := answerMap[propertyName]
	if value != "" {
		return true
	}
	return false
}

func getClusterIDFromClusterName(answerMap map[string]string) string {
	clusterName := answerMap["eksClusterName"]

	if clusterName != "" {
		return clusterName
	}

	return "xl-up-master"
}
