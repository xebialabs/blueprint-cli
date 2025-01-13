package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/internal/endpoints/v2"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"sort"
	"strconv"
	"unicode"

	"reflect"
	"strings"

	"github.com/xebialabs/blueprint-cli/pkg/models"
	"github.com/xebialabs/blueprint-cli/pkg/util"
)

const (
	Credentials = "credentials"
	Regions     = "regions"
)

type AWSFnResult struct {
	creds   aws.Credentials
	regions []string
}

func (result *AWSFnResult) GetResult(module string, attr string, index int) ([]string, error) {
	switch module {
	case Credentials:
		if attr == "" {
			return nil, fmt.Errorf("requested credentials attribute is not set")
		}

		// if requested, do exists check
		if attr == "IsAvailable" {
			return []string{strconv.FormatBool(result.creds.AccessKeyID != "")}, nil
		}

		// return attribute
		result, err := GetAWSCredentialsField(&result.creds, attr)
		if err == nil {
			return []string{result}, nil
		} else {
			return nil, err
		}
	case Regions:
		if index != -1 {
			return result.regions[index : index+1], nil
		}
		return result.regions, nil
	default:
		return nil, fmt.Errorf("%s is not a valid AWS module", module)
	}
}

// GetAvailableAWSRegions lists AWS regions for the service
func GetAvailableAWSRegionsForService(ctx context.Context, serviceID string) ([]string, error) {
	// Load the default configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS configuration: %w", err)
	}

	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(true), // Include all regions, not just ones enabled for this account
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe AWS regions: %w", err)
	}
	var regions []string
	for _, region := range output.Regions {
		regions = append(regions, *region.RegionName)
	}

	var availableRegions []string
	// Retrieve the custom endpoint resolver from the configuration
	resolver := cfg.EndpointResolverWithOptions

	// Check if resolver is nil
	if resolver == nil {
		return nil, fmt.Errorf("endpoint resolver is nil")
	}

	svc := ssm.New(cfg)

	// Check each region using the endpoint resolver
	for _, region := range regions {
		_, err := resolver.ResolveEndpoint(serviceID, region)
		if err == nil {
			availableRegions = append(availableRegions, region)
		}
	}

	sort.Strings(availableRegions)
	return availableRegions, nil
}

const ParamPath = "/aws/service/global-infrastructure/regions/%s/services"

func fetchEnabledServicesInRegion(ctx context.Context, region *string, sess *session.Session, cfg *aws.Config) (service map[string]bool, err error) {
	service = make(map[string]bool)
	svc := ssm.New(sess, cfg)
	var NextToken *string
next:
	resp, err := svc.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
		Path:      aws.String(fmt.Sprintf(ParamPath, *region)),
		NextToken: NextToken,
	})

	if err != nil {
		return
	}

	NextToken = resp.NextToken
	if len(resp.Parameters) > 0 {
		// fetch the service name, process
		for _, p := range resp.Parameters {
			if p.Value != nil {
				srv := formatServiceName(*p.Value)
				service[srv] = true
			}
		}
	}

	// AWS API call sometimes behave erratically, returning empty pages with NextToken as non null
	if NextToken != nil {
		goto next
	}
	return
}

func formatServiceName(s string) (name string) {
	name = strings.TrimSpace(s)

	// Replace all Non-letter/number values with space
	// AWS services are not named consistently
	runes := []rune(name)
	for i := 0; i < len(runes); i++ {
		if r := runes[i]; !(unicode.IsNumber(r) || unicode.IsLetter(r)) {
			runes[i] = ' '
		}
	}
	name = string(runes)
	// Title case name so its readable as a symbol.
	name = strings.ToLower(name)
	// Strip out spaces.
	name = strings.Replace(name, " ", "", -1)
	return
}

// GetAWSCredentialsFromSystem fetches stored AWS access keys from file or env keys
func GetAWSCredentialsFromSystem(ctx context.Context) (*aws.Credentials, error) {
	// Load the default configuration (looks for environment variables, profile config, etc.)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS configuration: %w", err)
	}

	// Retrieve credentials from the default credential provider
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve AWS credentials: %w", err)
	}

	return &creds, nil
}

func GetAWSCredentialsField(v *aws.Credentials, field string) (string, error) {
	r := reflect.ValueOf(v)

	f := r.Elem().FieldByName(field)
	if !f.IsValid() {
		return "", fmt.Errorf("field '%s' does not exist in aws.Credentials", field)
	}

	// Return the string value of the field
	if f.Kind() == reflect.String {
		return f.String(), nil
	}

	return "", fmt.Errorf("field '%s' is not a string type", field)
}

// CallAWSFuncByName calls related AWS module function with parameters provided
func CallAWSFuncByName(ctx context.Context, module string, params ...string) (models.FnResult, error) {
	switch strings.ToLower(module) {
	case Credentials:
		creds, err := GetAWSCredentialsFromSystem(ctx)
		if err != nil {
			// handle AWS configuration errors gracefully
			util.Verbose("[aws] Error while processing function [%s] is: %v\n", module, err)
			return &AWSFnResult{}, nil
		}
		return &AWSFnResult{creds: *creds}, nil
	case Regions:
		if len(params) < 1 || params[0] == "" {
			return nil, fmt.Errorf("service name parameter is required for AWS regions function")
		}
		regionsList, err := GetAvailableAWSRegionsForService(ctx, params[0])
		if err != nil {
			return nil, err
		}
		return &AWSFnResult{regions: regionsList}, err
	default:
		return nil, fmt.Errorf("%s is not a valid AWS module", module)
	}
}
