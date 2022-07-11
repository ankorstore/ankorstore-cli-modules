package util

import (
	"fmt"
	"os"
)

type Dirs struct {
	HomeDir   string
	AnkorDir  string
	ConfigDir string
	OptDir    string
	BinDir    string
	TmpDir    string
}

func NewDirs() Dirs {
	userHomeDir, _ := os.UserHomeDir()

	return Dirs{
		HomeDir: userHomeDir,
	}
}

func (d *Dirs) GetHomeDir() string {
	return d.HomeDir
}

func (d *Dirs) GetAnkorDir() string {
	return fmt.Sprintf("%s/.%s", d.GetHomeDir(), AppName)
}

func (d *Dirs) GetConfigDir() string {
	return fmt.Sprintf("%s/.%s/etc", d.GetHomeDir(), AppName)
}

func (d *Dirs) GetOptDir() string {
	return fmt.Sprintf("%s/.%s/opt", d.GetHomeDir(), AppName)
}

func (d *Dirs) GetBinDir() string {
	return fmt.Sprintf("%s/.%s/bin", d.GetHomeDir(), AppName)
}

func (d *Dirs) GetTmpDir() string {
	return fmt.Sprintf("%s/.%s/tmp", d.GetHomeDir(), AppName)
}

func (d *Dirs) GetPluginsDir() string {
	return fmt.Sprintf("%s/.%s/plugins", d.GetHomeDir(), AppName)
}

func (d *Dirs) GetLogsDir() string {
	return fmt.Sprintf("%s/.%s/logs", d.GetHomeDir(), AppName)
}
