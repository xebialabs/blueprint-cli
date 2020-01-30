package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/gobuffalo/packr"
	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/pkg/util"
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
		util.Fatal("No license data found")
	}
	sort.Strings(all)
	fmt.Fprintf(out, "## Licenses information\n\n")
	fmt.Fprintf(out, "This notice contains information about the OSS software used in the XL CLI.\n")
	fmt.Fprintf(out, "The XL CLI uses the following Open source components. Visit the respective URL to view their licence\n")
	fmt.Fprintf(out, "--------------------------------------------------------------------------------------------------------\n\n")
	for _, lic := range all {
		s, _ := box.FindString(lic)
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
		return substr(licName, 0, len(licName)-extLength)
	} else {
		return licName
	}
}

func substr(src string, offset int, length int) string {
	runeS := []rune(src)
	sub := runeS[offset:length]
	return string(sub)
}
