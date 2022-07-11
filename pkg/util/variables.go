package util

import (
	"embed"
)

var (
	AppName    = "ankor"
	TemplateFS embed.FS
	PluginFs   embed.FS
	confDir    string
	Force      bool
)

const (
	Homebrew = "homebrew"
)

func init() {
	dirs := NewDirs()
	confDir = dirs.GetConfigDir()
}
