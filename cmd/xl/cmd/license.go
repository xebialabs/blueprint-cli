package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
	"strings"
	"os"
	"github.com/gobuffalo/packr"
	"sort"
	"io"
	"github.com/xebialabs/xl-cli/pkg/xl"
)

var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Display license info",
	Long:  `Display license info`,
	Run: func(cmd *cobra.Command, args []string) {
		PrintLicenses(os.Stdout)
	},
}

func PrintLicenses(out io.Writer) {
	box := packr.NewBox("../../../licenses")
	all := box.List()
	if len(all) == 0 {
		xl.Fatal("No license data found")
	}
	sort.Strings(all)
	fmt.Fprintf(out, "## Licenses information\n\n")
	fmt.Fprintf(out, "This notice contains information about the OSS software used in the XL Cli.\n")
	fmt.Fprintf(out, "The XL Cli contains the following OSS components: \n\n")
	for _, licName := range all {
		fmt.Fprintf(out, " * %s\n", exandName(licName))
	}
	fmt.Fprintf(out, "\nThe full text of the licenses are:\n")
	for _, lic := range all {
		s, _ := box.FindString(lic)
		fmt.Fprintf(out, "\n--------------------------------------------------------------------------------\n")
		fmt.Fprintf(out, "## %s\n\n", cutExtension(lic))
		fmt.Fprintf(out, s)
	}
}

func init() {
	rootCmd.AddCommand(licenseCmd)
}

func exandName(licName string) string {
	name := cutExtension(licName)
	splitted := strings.SplitN(name, "-", 2)
	if len(splitted) >= 2 {
		return fmt.Sprintf("%s - http://github.com/%s/%s", name, splitted[0], splitted[1])
	} else {
		return "unknown source"
	}
}

func cutExtension(licName string) string {
	extLength := len("-license.MD")
	if len(licName) > extLength {
		return substr(licName, 0, len(licName) - extLength)
	} else {
		return licName
	}
}

func substr(src string, offset int, length int) string {
	runeS := []rune(src)
	sub := runeS[offset:length]
	return string(sub)
}