package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEmptyIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	basedir := "../.."
	dirname := filepath.Join(basedir, "src/picasa-dl/test/empty_index")
	cmd := exec.Command(filepath.Join(basedir, "bin/picasa-dl"), "-d="+dirname)
	err := cmd.Run()
	if err != nil {
		t.Log(err)
	}
	filename := filepath.Join(dirname, "sample.user/albums/index.html")
	fi, err := os.Stat(filename)
	if err != nil {
		t.Error(err)
	}
	if fi.Size() != 1524 {
		t.Error(filename, "file size:", fi.Size())
	}
}
