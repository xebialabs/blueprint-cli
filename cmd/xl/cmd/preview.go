package cmd

import (
	"github.com/disiqueira/gotree"
	"github.com/spf13/cobra"
	"github.com/xebialabs/xl-cli/pkg/models"
	"github.com/xebialabs/xl-cli/pkg/util"
	"github.com/xebialabs/xl-cli/pkg/xl"
	"path/filepath"
)

var previewFilenames []string
var previewValues map[string]string

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview Deployment",
	Long:  `Preview Deployment plan`,
	Run: func(cmd *cobra.Command, args []string) {
		DoPreview(previewFilenames)
	},
}

func printTaskItem(t *gotree.Tree, item models.TaskPreviewItem) {
	current := (*t).Add(item.Name)
	for _, i := range item.Children {
		printTaskItem(&current, i)
	}
}

func printTask(info *models.TaskInfo) {
	if info != nil {
		tree := gotree.New(info.Description)
		for _, item := range info.Steps {
			printTaskItem(&tree, item)
		}
		util.Info(tree.Print())
	}
}

func printPreview(response *models.PreviewResponse) {
	if response != nil {
		printTask(response.Task)
	}
}

func previewDocument(context *xl.Context, fileWithDocs xl.FileWithDocuments, doc *xl.Document, _ bool) {
	previewDir := filepath.Dir(fileWithDocs.FileName)
	preview, err := context.PreviewSingleDocument(doc, previewDir)
	if err != nil {
		xl.ReportFatalDocumentError(fileWithDocs.FileName, doc, err)
	} else {
		printPreview(preview)
	}
}

func DoPreview(previewFilenames []string) {
	xl.ForEachDocument("Previewing", previewFilenames, previewValues, false, previewDocument)
}

func init() {
	rootCmd.AddCommand(previewCmd)

	previewFlags := previewCmd.Flags()
	previewFlags.StringArrayVarP(&previewFilenames, "file", "f", []string{}, "Path(s) to the file(s) to apply (required)")
	_ = applyCmd.MarkFlagRequired("file")
	previewFlags.StringToStringVar(&previewValues, "values", map[string]string{}, "Values")
}
