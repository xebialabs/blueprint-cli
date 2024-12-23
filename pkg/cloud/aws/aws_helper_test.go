package aws

import (
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "strconv"
    "testing"
)

func TestGetAWSCredentialsFromSystem(t *testing.T) {
    t.Run("should get credentials file from the underlying system", func(t *testing.T) {
        result, err := CallAWSFuncByName(context.TODO(), Credentials)
        require.Nil(t, err)
        values, err := result.GetResult(Credentials, "ProviderName", -1)
        require.Nil(t, err)
        assert.NotEmpty(t, values)
    })

    t.Run("should return bool result of aws.credentials().IsAvailable", func(t *testing.T) {
        result, err := CallAWSFuncByName(context.TODO(), Credentials)
        require.Nil(t, err)
        values, err := result.GetResult(Credentials, "IsAvailable", -1)
        require.Nil(t, err)
        assert.Len(t, values, 1)
        _, err = strconv.ParseBool(values[0])
        require.Nil(t, err)
    })

    t.Run("should error when credentials attribute is not set", func(t *testing.T) {
        result, err := CallAWSFuncByName(context.TODO(), Credentials)
        require.Nil(t, err)
        _, err = result.GetResult(Credentials, "", -1)
        require.NotNil(t, err)
        assert.Equal(t, "requested credentials attribute is not set", err.Error())
    })

    t.Run("should get list of regions for AWS ECS service", func(t *testing.T) {
        result, err := CallAWSFuncByName(context.TODO(), Regions, "ecs")
        require.Nil(t, err)
        values, err := result.GetResult(Regions, "", -1)
        require.Nil(t, err)
        assert.NotEmpty(t, values)
        assert.True(t, len(values) > 1)
    })

    t.Run("should get first region for AWS ECS service", func(t *testing.T) {
        result, err := CallAWSFuncByName(context.TODO(), Regions, "ecs")
        require.Nil(t, err)
        values, err := result.GetResult(Regions, "", 1)
        require.Nil(t, err)
        assert.NotEmpty(t, values)
        assert.Len(t, values, 1)
    })

    t.Run("should error on missing AWS service parameter", func(t *testing.T) {
        _, err := CallAWSFuncByName(context.TODO(), Regions)
        require.NotNil(t, err)
        assert.Equal(t, "service name parameter is required for AWS regions function", err.Error())
    })
}
