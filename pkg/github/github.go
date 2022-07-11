package github

import (
	"context"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/util"
	"github.com/google/go-github/v44/github"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type GithubContainer struct {
	Client   *github.Client
	Releases []*github.RepositoryRelease
}

func (gh *GithubContainer) getGithubClient() *github.Client {
	if gh.Client != nil {
		return gh.Client
	}

	tkn := viper.GetString("git.github.token")
	if tkn == "" {
		log.Fatal().Msgf("Github token not found in config. Please run `%s init ` without arguments to re-initialise %s", util.AppName, util.AppName)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: tkn},
	)
	tc := oauth2.NewClient(ctx, ts)
	gh.Client = github.NewClient(tc)
	return gh.Client
}
