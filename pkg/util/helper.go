package util

import (
	"os/exec"
	"strings"
)

func IsBrewInstallation() bool {
	ankorPath, _ := exec.LookPath(AppName)
	if strings.Contains(ankorPath, Homebrew) {
		return true
	}
	return false
}
