//go:build linux || darwin

package template

import (
	"io/fs"
	"log"
	"os"
	"syscall"
)

func chown(dest *os.File, fi fs.FileInfo) {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		if err := dest.Chown(int(stat.Uid), int(stat.Gid)); err != nil {
			log.Fatalf("Unable to chown temp file: %s\n", err)
		}
	}
}
