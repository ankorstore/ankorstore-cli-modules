package docker

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ankorstore/ankorstore-cli-modules/pkg/errorhandling"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/go-errors/errors"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/jhoonb/archivex"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	StatusOutput OutputKey  = "status"
	StreamOutput OutputKey  = "stream"
	Gcloud       AuthHelper = "gcloud"
)

var (
	containers client.ContainerAPIClient
	builder    client.ImageAPIClient
	ctx        context.Context
	auth       authn.Authenticator
)

type OutputKey string
type AuthHelper = string

func init() {
	var err error
	ctx = context.Background()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	errorhandling.CheckFatal(err, "X Error connecting to docker ")
	containers = dockerClient
	builder = dockerClient
}

func IsDockerRunning() (bool, error) {
	_, err := containers.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "Cannot connect") || strings.Contains(err.Error(), "connection refused") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func getGcloudAuthConfig() (*authn.AuthConfig, error) {
	if auth == nil {
		var err error
		auth, err = google.NewGcloudAuthenticator()
		errorhandling.CheckFatal(err, "X Error configuring gcloud authenticator ")
	}

	authConfig, err := auth.Authorization()
	if err != nil {
		return &authn.AuthConfig{}, err
	}
	return authConfig, nil
}

func getGcloudAuthString() (string, error) {
	authConfig, err := getGcloudAuthConfig()
	if err != nil {
		return "", err
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	return authStr, nil
}

func PullImage(image string) error {
	var options = types.ImagePullOptions{}

	if viper.InConfig("docker.authentication") {
		for _, a := range viper.GetStringSlice("docker.authentication") {
			switch a {
			case Gcloud:
				authStr, err := getGcloudAuthString()
				if err != nil {
					return errors.Wrap(err, 0)
				}
				options.RegistryAuth = authStr
			}
		}
	}

	response, err := builder.ImagePull(ctx, image, options)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	return printOutput(response, StatusOutput)
}

func GetAuthConfig() (map[string]types.AuthConfig, error) {
	var authConfig = map[string]types.AuthConfig{}
	if viper.InConfig("docker.authentication") {
		for _, a := range viper.GetStringSlice("docker.authentication") {
			switch a {
			case Gcloud:
				authData, err := getGcloudAuthConfig()
				if err != nil {
					return authConfig, err
				}
				var authc = types.AuthConfig{
					Username: "oauth2accesstoken",
					Password: authData.Password,
				}

				authConfig["eu.gcr.io"] = authc
				authConfig["gcr.io"] = authc
				authConfig["asia.gcr.io"] = authc
				authConfig["eu.gcr.io"] = authc
				authConfig["gcr.io"] = authc
				authConfig["marketplace.gcr.io"] = authc
				authConfig["staging-k8s.gcr.io"] = authc
				authConfig["us.gcr.io"] = authc
			}
		}
	}
	return authConfig, nil
}

func BuildImage(tag, path, dockerfile string) error {
	log.Debug().Msgf("Running the equivalent of `docker build -t %s -f %s/%s %s", tag, path, dockerfile, path)

	authMap, err := GetAuthConfig()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	opts := types.ImageBuildOptions{
		Dockerfile:  dockerfile,
		Tags:        []string{tag},
		ForceRemove: true,
		PullParent:  true,
		AuthConfigs: authMap,
		//Version: types.BuilderBuildKit,
	}

	ctxFile, err := createContextFile(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer func() { _ = ctxFile.Close() }()

	response, err := builder.ImageBuild(context.TODO(), ctxFile, opts)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer func() { _ = response.Body.Close() }()

	return printOutput(response.Body, StreamOutput)
}

func createContextFile(ctx string) (*os.File, error) {
	randInt, err := rand.Int(rand.Reader, big.NewInt(27))
	if err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("/tmp/ankor-%d.tar", randInt)
	tar := new(archivex.TarFile)
	_ = tar.Create(filename)
	_ = tar.AddAll(ctx, false)
	_ = tar.Close()
	return os.Open(filename)
}

func printOutput(reader io.Reader, key OutputKey) error {
	type OutputLine struct {
		Stream      string `json:"stream"`
		Status      string `json:"status"`
		ErrorDetail struct {
			Message string `json:"message"`
		} `json:"errorDetail"`
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var line OutputLine
		t := scanner.Text()
		errorhandling.CheckError(json.Unmarshal([]byte(t), &line))
		var out []string
		switch key {
		case StreamOutput:
			out = strings.Split(line.Stream, "\n")
		case StatusOutput:
			out = strings.Split(line.Status, "\n")
		}
		if len(line.ErrorDetail.Message) > 0 {
			out = strings.Split(line.ErrorDetail.Message, "\n")
		}
		for _, l := range out {
			if len(l) > 0 {
				log.Debug().Msgf("\t| %s", l)
			}
		}
	}
	return nil
}

func Run(opts ...RunOpt) error {
	runConfig := &RunConfig{}
	for _, o := range opts {
		err := o(runConfig)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}
	log.Info().
		Str("image", runConfig.Config.Image).
		Str("entrypoint", strings.Join(runConfig.Config.Entrypoint, " ")).
		Str("cmd", strings.Join(runConfig.Config.Cmd, " ")).
		Msg("Running container")

	resp, err := containers.ContainerCreate(ctx,
		runConfig.Config,
		runConfig.HostConfig,
		runConfig.NetworkConfig,
		runConfig.Platform,
		runConfig.Name)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	if err := containers.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, 0)
	}
	statusCh, errCh := containers.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return errors.Wrap(err, 0)
		}
	case <-statusCh:
	}
	out, err := containers.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return errors.Wrap(err, 0)
	}
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
