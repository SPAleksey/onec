package cmd

import (
	"os"
)

func CheckFlag(PathToBase *string) {

	if *PathToBase == "" {
		path, err := os.Executable()
		if err != nil {
			panic("can't find folder")
		}
		path = path + "\\1Cv8.1CD"
		*PathToBase = path
	}
}
