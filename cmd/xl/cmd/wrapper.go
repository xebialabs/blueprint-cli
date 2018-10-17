package cmd

//go:generate $GOPATH/bin/templify -p wrapper -e -o wrapper/xlw.go wrapper/source/xlw
//go:generate $GOPATH/bin/templify -p wrapper -e -o wrapper/xlwBat.go wrapper/source/xlw.bat
//go:generate $GOPATH/bin/templify -p wrapper -e -o wrapper/wrapper.conf.go wrapper/source/wrapper.conf

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/cmd/xl/cmd/wrapper"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
)

type WrapperConfig struct {
	CliVersion        string
	CliBaseUrl        string
	WrapperConfigName string
}

var CliBaseUrl = "https://s3.amazonaws.com/xl-cli/bin"

func writeTemplateFile(name string, templateString string, config WrapperConfig) error {
	tpl, err := template.New(name).Parse(templateString)
	if err != nil {
		return err
	}
	file, err := os.Create(name)
	defer file.Close()
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		err = file.Chmod(os.ModePerm)
		if err != nil {
			return err
		}
	}
	return tpl.Execute(file, config)
}

func writeWrapperScripts(config WrapperConfig) error {
	err := writeTemplateFile("xlw.bat", wrapper.XlwBatTemplate(), config)
	if err != nil {
		return err
	}
	return writeTemplateFile("xlw", wrapper.XlwTemplate(), config)
}

func writeConfigFile(config WrapperConfig) error {
	var confDir = filepath.FromSlash("xl/wrapper")
	var confFile = filepath.FromSlash(fmt.Sprintf("%s/%s", confDir, config.WrapperConfigName))
	os.MkdirAll(confDir, os.ModePerm)
	return writeTemplateFile(confFile, wrapper.WrapperTemplate(), config)
}

func writeFiles(config WrapperConfig) error {
	var err = writeWrapperScripts(config)
	if err != nil {
		return err
	}
	err = writeConfigFile(config)
	if err != nil {
		return err
	}
	return nil
}

var wrapperCmd = &cobra.Command{
	Use:   "wrapper",
	Short: "Generate XL wrapper",
	Long:  "Generate XL wrapper files and configuration",
	Run: func(cmd *cobra.Command, args []string) {
		xl.Verbose("Generating wrapper files... ")
		var err = writeFiles(WrapperConfig{
			CliVersion:        CliVersion,
			CliBaseUrl:        CliBaseUrl,
			WrapperConfigName: "wrapper.conf",
		})
		if err != nil {
			xl.Fatal("\nError creating wrapper files: %s\n", err)
		} else {
			xl.Verbose("done.\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(wrapperCmd)
}
