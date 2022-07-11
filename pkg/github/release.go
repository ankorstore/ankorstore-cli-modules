package github

import (
	"context"
	"fmt"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/filesystem"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/util"
	"github.com/go-errors/errors"
	"net/http"
	"strings"

	"github.com/google/go-github/v44/github"
)

var (
	ErrCouldNotUpdate = errors.New(fmt.Sprintf("could not update %s", util.AppName))
)

func (gh *GithubContainer) loadReleases() error {
	var err error
	if gh.Releases != nil && len(gh.Releases) > 0 {
		return nil
	}

	githubClient := gh.getGithubClient()
	gh.Releases, _, err = githubClient.Repositories.ListReleases(context.Background(), "ankorstore", "ankorstore-cli", &github.ListOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (gh *GithubContainer) GetLatestRelease() (*github.RepositoryRelease, error) {
	err := gh.loadReleases()
	if err != nil {
		return nil, err
	}

	for _, rel := range gh.Releases {
		if !rel.GetDraft() {
			return rel, nil
		}
	}
	return nil, nil
}

func (gh *GithubContainer) GetChangeLog(currentVersion string) (string, error) {
	err := gh.loadReleases()
	if err != nil {
		return "", err
	}

	var changelog []string

	for _, rel := range gh.Releases {
		if strings.TrimLeft(rel.GetTagName(), "v") == currentVersion {
			break
		}
		changelog = append([]string{fmt.Sprintf("%s\n%s\n", rel.GetTagName(), rel.GetBody())}, changelog...)
	}

	return strings.Join(changelog, ""), nil
}

// DownloadAsset downloads asset from a release
func (gh *GithubContainer) DownloadAsset(asset *github.ReleaseAsset, targetPath string) error {
	githubClient := gh.getGithubClient()
	assetReader, _, err := githubClient.Repositories.DownloadReleaseAsset(context.Background(), "ankorstore", "ankorstore-cli", asset.GetID(), &http.Client{})
	if err != nil {
		return fmt.Errorf("failed to download github asset: %v", err)
	}
	defer assetReader.Close()
	if assetReader != nil {
		err = filesystem.SaveBinaryFile(targetPath, assetReader)
		if err != nil {
			return fmt.Errorf("failed to store github asset: %v", err)
		}
		return nil
	}
	return errors.Wrap(ErrCouldNotUpdate, 0)
}
