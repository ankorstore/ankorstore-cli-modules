package util

import (
	"bufio"
	"fmt"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/errorhandling"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-errors/errors"
	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog/log"
)

var (
	ErrMissingExecutable = errors.New("must be installed on your system and available in your $PATH")
)

type DependencyInstaller func() (string, error)

// GetPath attempts to retrieve path for specified executable name.
func GetPath(name string, installers ...DependencyInstaller) string {
	path, err := exec.LookPath(name)
	if err != nil {
		if len(installers) > 0 {
			for _, install := range installers {
				if path, err = install(); err == nil {
					return path
				}
			}
		}
		errorhandling.CheckFatal(errors.New(fmt.Errorf("'%s' %w", name, ErrMissingExecutable)))
	}
	log.Debug().Msgf("Found executable for '%s' at %s", name, path)
	return path
}

// AddToPathCmd will use the supplied cmd to retrieve the value for a path
// and add the sub suffix to the pat for adding to PATH.
func AddToPathCmd(sub string, cmd ...string) (string, error) {
	c := exec.Command(cmd[0], cmd[1:]...) // nolint: gosec
	out, err := c.CombinedOutput()
	if err != nil {
		return "", err
	}
	target := strings.Trim(string(out), " \n")
	sep := string(filepath.Separator)
	return AddToPath(fmt.Sprintf("%s%s%s", strings.TrimRight(target, sep), sep, sub))
}

// AddToPath attempts to add supplied path to the systems shell environment
// as well as adding it to the os.Environ for teh current run.
func AddToPath(path string) (string, error) {
	homeDir, _ := os.UserHomeDir()
	right := strings.TrimRight(homeDir, string(filepath.Separator))
	pathWithHome := strings.Replace(path, right, "$HOME", -1)

	targets := []string{
		fmt.Sprintf("%s/.profile", homeDir),
		fmt.Sprintf("%s/.zshrc", homeDir),
		fmt.Sprintf("%s/.bashrc", homeDir),
	}

	templates := map[string]string{
		targets[0]: fmt.Sprintf(`
# added by %s
if [ -d "%s" ] ; then
   PATH="%s:$PATH"
fi
`, AppName, pathWithHome, pathWithHome),
		targets[1]: fmt.Sprintf(`
# added by %s
export PATH="%s:$PATH"
`, AppName, pathWithHome),
		targets[2]: fmt.Sprintf(`
# added by %s
if [ -d "%s" ] ; then
   export PATH="%s:$PATH"
fi
`, AppName, pathWithHome, pathWithHome),
	}

	if runtime.GOOS == "darwin" {
		zshrc := fmt.Sprintf("%s/.zshrc", homeDir)
		if _, err := os.Stat(zshrc); os.IsNotExist(err) {
			emptyFile, err := os.Create(zshrc)
			if err != nil {
				return "", errors.Wrap(err, 0)
			}
			_ = emptyFile.Close()
		}
	}

	var inPath = false

	for _, target := range targets {
		if _, err := os.Stat(target); os.IsNotExist(err) || inPath {
			continue
		}

		log.Debug().Msgf("Checking %s for suitable path entry containing %s", target, pathWithHome)

		f, err := os.Open(target)
		if err != nil {
			return path, errors.Wrap(err, 0)
		}

		// Splits on newlines by default.
		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			// todo: should be more complex a check, i.e regex for comments, etc
			if strings.Contains(scanner.Text(), pathWithHome) {
				inPath = true
			}
		}

		if err := scanner.Err(); err != nil {
			return path, errors.Wrap(err, 0)
		}

		_ = f.Close()
		if !inPath {
			log.Debug().Msgf("✘ No Entry found: adding PATH entry to %s for %s", target, pathWithHome)

			prompt := promptui.Select{
				Label: fmt.Sprintf("Would you like to save PATH entry to %s for %s", target, pathWithHome),
				Items: []string{"Yes", "No"},
			}
			_, result, _ := prompt.Run()
			if "No" == result {
				break
			} else {
				// add to file
				f, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return path, errors.Wrap(err, 0)
				}

				if _, err := f.WriteString(templates[target]); err != nil {
					return path, errors.Wrap(err, 0)
				}
				_ = f.Close()

				// refresh the shell environment
				log.Info().Msgf(`Please make sure you run (source %s to refresh the %s)`, target, target)
				break
			}
		} else {
			log.Debug().Msgf(`✔ Entry found in %s for %s`, target, pathWithHome)
			break
		}
	}

	if !inPath {
		_ = os.Setenv("PATH", fmt.Sprintf("%s:%s", path, os.Getenv("PATH")))
	}

	return path, nil
}
