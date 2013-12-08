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
	filename := filepath.Join(dirname, "html/index.html")
	fi, err := os.Stat(filename)
	if err != nil {
		t.Error(err)
	}
	if fi.Size() != 1540 {
		t.Error(filename, "file size:", fi.Size())
	}
}

func TestLANGUAGE(t *testing.T) {
	language := os.Getenv("LANGUAGE")
	defer os.Setenv("LANGUAGE", language)

	os.Setenv("LANGUAGE", "ja_JP.UTF-8")
	lang := getLANGUAGE()
	if lang != "ja" {
		t.Error("is not `ja`: ", lang)
	}
}
