package handle

import (
	"github.com/xebialabs/xl-cli/internal/platform/files"
	"log"
	"os"
)

func CloseFiles(fls []*os.File) {
	err := files.Close(fls)

	if len(err) > 0 {
		log.Println("error closing files:", err)
	}
}
