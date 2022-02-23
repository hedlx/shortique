package main

import (
	"os"
	"path/filepath"
)

func Setup() error {
	TmpDir = filepath.FromSlash(TmpDir)

	if err := os.RemoveAll(TmpDir); err != nil {
		return err
	}

	if err := os.Mkdir(TmpDir, 0777); err != nil {
		return err
	}

	return nil
}
