package utils

import (
	"os"
	"path/filepath"
)

func GetFullArgv0() (string, error) {
	abs, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)

}
