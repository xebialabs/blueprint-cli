package blueprint

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xebialabs/xl-blueprint/pkg/util"
)

func TestGeneratedBlueprintRegistersCreatedFile(t *testing.T) {
	var gb GeneratedBlueprint
	defer gb.Cleanup()

	util.IsVerbose = true

	file := "foo.tmp"
	assert.False(t, exists(file))
	_, err := gb.GetOutputFile(file)
	assert.Nil(t, err)
	assert.FileExists(t, file)
	assert.Contains(t, gb.GeneratedFiles, file)
}
func TestGeneratedBlueprintRegistersCreatedDirectory(t *testing.T) {
	util.IsVerbose = true

	var gb GeneratedBlueprint
	defer gb.Cleanup()
	file := "foo/foo.tmp"
	assert.False(t, exists(file))
	assert.False(t, exists(filepath.Dir(file)))
	_, err := gb.GetOutputFile(file)
	assert.Nil(t, err)
	assert.FileExists(t, file)
	assert.Contains(t, gb.GeneratedFiles, filepath.Dir(file))
	assert.Contains(t, gb.GeneratedFiles, file)
}

func TestGeneratedBlueprintCreatesAllIntermediateDirectories(t *testing.T) {
	util.IsVerbose = true
	var gb GeneratedBlueprint
	defer gb.Cleanup()
	file := "foo/bar/foo.tmp"
	_, err := gb.GetOutputFile(file)
	assert.Nil(t, err)
	assert.FileExists(t, file)
	assert.Contains(t, gb.GeneratedFiles, filepath.Dir(file))
	assert.Contains(t, gb.GeneratedFiles, filepath.Dir(filepath.Dir(file)))
	assert.Contains(t, gb.GeneratedFiles, file)
}
func TestGeneratedBlueprintDoesNotRegisterExistingDirectory(t *testing.T) {
	os.Mkdir("foo", os.ModePerm)
	var gb GeneratedBlueprint
	file := "foo/bar/foo.tmp"
	_, err := gb.GetOutputFile(file)
	assert.Nil(t, err)
	assert.FileExists(t, file)
	assert.Contains(t, gb.GeneratedFiles, filepath.Dir(file))
	assert.Contains(t, gb.GeneratedFiles, file)
	assert.NotContains(t, gb.GeneratedFiles, "foo")
	gb.Cleanup()
	os.Remove("foo")
}

func TestCreateDirectoryIfNeeded(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "pathTest")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)
	defer os.RemoveAll("test")
	type args struct {
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"create directory if it doesn't exist", args{path.Join("test", "test.xlval")}, false},
		{"Do not create directory if it exists", args{path.Join(tmpDir, "test.xlval")}, false},
		{"Do not do anything if there is no directory", args{"test.xlval"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gb := new(GeneratedBlueprint)
			if err := gb.createDirectoryIfNeeded(path.Dir(tt.args.fileName)); (err != nil) != tt.wantErr {
				t.Errorf("createDirectoryIfNeeded() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
