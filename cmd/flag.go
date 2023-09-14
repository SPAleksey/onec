package cmd

import (
	"os"
	"path/filepath"
)

func CheckFlag(PathToBase *string) {

	if *PathToBase == "" {
		path, err := os.Executable()
		if err != nil {
			panic("can't find folder")
		}
		path = filepath.Join(filepath.Dir(path), "1Cv8.1CD")
		*PathToBase = path
	}
}
