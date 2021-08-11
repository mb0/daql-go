package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

func GoTool(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	return cmd.Output()
}

func GoModPath(dir string) (string, error) {
	b, err := GoTool(dir, "list", "-m")
	if err != nil {
		return "", fmt.Errorf("go mod path for %s: %v", dir, err)
	}
	return strings.TrimSpace(string(b)), nil
}
