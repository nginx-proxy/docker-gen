//go:build windows

package template

import (
	"io/fs"
	"os"
)

func chown(dest *os.File, fi fs.FileInfo) {
	// do nothing
}
