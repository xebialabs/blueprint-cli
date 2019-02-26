package aws

import (
	"reflect"
	"testing"

	"github.com/xebialabs/xl-cli/pkg/models"
)

func TestK8SFnResult_GetResult(t *testing.T) {
	type fields struct {
		cluster K8sCluster
		context K8sContext
		user    K8sUser
	}
	type args struct {
		module string
		attr   string
		index  int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &K8SFnResult{
				cluster: tt.fields.cluster,
				context: tt.fields.context,
				user:    tt.fields.user,
			}
			got, err := result.GetResult(tt.args.module, tt.args.attr, tt.args.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("K8SFnResult.GetResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("K8SFnResult.GetResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getK8SConfigField(t *testing.T) {
	type args struct {
		v     *K8SFnResult
		field string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getK8SConfigField(tt.args.v, tt.args.field); got != tt.want {
				t.Errorf("getK8SConfigField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCallK8SFuncByName(t *testing.T) {
	type args struct {
		module string
		params []string
	}
	tests := []struct {
		name    string
		args    args
		want    models.FnResult
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CallK8SFuncByName(tt.args.module, tt.args.params...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CallK8SFuncByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CallK8SFuncByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetK8SConfigFromSystem(t *testing.T) {
	type args struct {
		context string
	}
	tests := []struct {
		name    string
		args    args
		want    K8SFnResult
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetK8SConfigFromSystem(tt.args.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetK8SConfigFromSystem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetK8SConfigFromSystem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetKubeConfigFile(t *testing.T) {
	tests := []struct {
		name    string
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKubeConfigFile()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKubeConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKubeConfigFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseKubeConfig(t *testing.T) {
	type args struct {
		kubeConfigYaml []byte
	}
	tests := []struct {
		name    string
		args    args
		want    K8sConfig
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKubeConfig(tt.args.kubeConfigYaml)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKubeConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseKubeConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetContext(t *testing.T) {
	type args struct {
		config  K8sConfig
		context string
	}
	tests := []struct {
		name    string
		args    args
		want    K8SFnResult
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetContext(tt.args.config, tt.args.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
