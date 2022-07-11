package utils

import (
	"fmt"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/errorhandling"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/util"
	"github.com/go-errors/errors"
	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var (
	ErrEmptyGithubToken = fmt.Errorf("PAT must be provided, generate one at https://github.com/settings/tokens")
)

type GithubHelper struct {
	Flags   *pflag.FlagSet
	ConfDir string
}

func NewGithubHelper(flags *pflag.FlagSet, confDir string) *GithubHelper {
	return &GithubHelper{
		Flags:   flags,
		ConfDir: confDir,
	}
}

func (gh *GithubHelper) SetupGithub() (string, string, error) {
	var githubUser, githubToken string
	var err error

	githubUser = os.Getenv("HOMEBREW_GITHUB_USERNAME")
	githubToken = os.Getenv("HOMEBREW_GITHUB_API_TOKEN")

	if githubUser == "" || githubToken == "" {
		githubUser, err = gh.Flags.GetString("githubuser")
		if err != nil {
			return "", "", errors.Wrap(err, 0)
		}
		githubToken, err = gh.Flags.GetString("githubtoken")
		if err != nil {
			return "", "", errors.Wrap(err, 0)
		}
	}

	// if global ankor.yaml file exists pull values from it and copy them into the new file
	if gh.isGlobalScopeConfigured(gh.ConfDir) {
		if githubUser == "" {
			githubUser = viper.GetString("git.github.user")
		}
		if githubToken == "" {
			githubToken = viper.GetString("git.github.token")
		}
	}

	if githubUser == "" || githubToken == "" {
		log.Debug().Msgf("Missing authentication details for Github: https://github.com/settings/tokens")
		log.Info().Msgf("For more info on creating a personal Access Token see \n https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token ")

		prompt := promptui.Prompt{
			Label:   "Github Username",
			Default: githubUser,
		}
		githubUser, err = prompt.Run()
		errorhandling.CheckFatal(err)

		validateToken := func(input string) error {
			if len(input) == 0 {
				return errors.Wrap(ErrEmptyGithubToken, 0)
			}
			return nil
		}

		prompt = promptui.Prompt{
			Label:    "Github Personal Access Token",
			Default:  githubToken,
			Mask:     '*',
			Validate: validateToken,
		}
		githubToken, err = prompt.Run()
		errorhandling.CheckFatal(err)
	}

	return githubUser, githubToken, nil
}

// isGlobalScopeConfigured checks that the global ankor.yaml file exists.
func (gh *GithubHelper) isGlobalScopeConfigured(confDir string) bool {
	path := filepath.Join(confDir, fmt.Sprintf("%s.yaml", util.AppName))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
