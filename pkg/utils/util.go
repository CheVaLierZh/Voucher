package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ExecFileDir() string {
	path, _ := exec.LookPath(os.Args[0])
	path, _ = filepath.Abs(path)
	dir := path[:strings.LastIndex(path, string(os.PathSeparator))]
	return dir
}
