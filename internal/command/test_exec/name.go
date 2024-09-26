package test_exec

import (
	"fmt"
	"strings"
)

func splitName(name string) (testName, execName string, err error) {
	idx := strings.LastIndex(name, "-")
	if idx == -1 || idx == len(name)-1 {
		return "", "", fmt.Errorf("invalid test execution name: %q", name)
	}
	testName = name[:idx]
	execName = name
	return
}
