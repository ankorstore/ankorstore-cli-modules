package docker

import (
	"github.com/ankorstore/ankorstore-cli-modules/pkg/docker/mocks"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/go-errors/errors"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/phpboyscout/zltest"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestIsDockerRunning(t *testing.T) {
	t.Run("with running docker daemon", func(t *testing.T) {
		c := &mocks.ContainerAPIClient{}
		c.On("ContainerList", mock.Anything, mock.Anything).Once().Return(nil, nil)
		containers = c
		running, err := IsDockerRunning()
		assert.NoError(t, err)
		assert.True(t, running)
	})

	t.Run("with stopped docker daemon", func(t *testing.T) {
		c := &mocks.ContainerAPIClient{}
		c.On("ContainerList", mock.Anything, mock.Anything).
			Once().
			Return(nil,
				errors.New("random string with 'Cannot connect' in the middle"))
		containers = c
		running, err := IsDockerRunning()
		assert.NoError(t, err)
		assert.False(t, running)
	})

	t.Run("with starting/stopping docker daemon", func(t *testing.T) {
		c := &mocks.ContainerAPIClient{}
		c.On("ContainerList", mock.Anything, mock.Anything).
			Once().
			Return(nil,
				errors.New("random string with 'connection refused' in the middle"))
		containers = c
		running, err := IsDockerRunning()
		assert.NoError(t, err)
		assert.False(t, running)
	})

	t.Run("with unknown docker state with error", func(t *testing.T) {
		c := &mocks.ContainerAPIClient{}
		c.On("ContainerList", mock.Anything, mock.Anything).
			Once().
			Return(nil,
				errors.New("a generic error message"))
		containers = c
		running, err := IsDockerRunning()
		assert.Error(t, err)
		assert.False(t, running)
	})
}

func TestGetAuthConfig(t *testing.T) {
	t.Run("with valid gcloud authentication", func(t *testing.T) {
		cfg := `{"docker": { "authentication" : ["gcloud"]}}`
		viper.Reset()
		viper.SetConfigType("json")
		_ = viper.ReadConfig(strings.NewReader(cfg))

		authMock := &mocks.Authenticator{}
		authMock.On("Authorization").
			Return(&authn.AuthConfig{
				Username: "_token",
				Password: "ya29.encryptedtoken",
			}, nil)

		auth = authMock

		result, err := GetAuthConfig()
		assert.NoError(t, err)

		assert.Len(t, result, 6)
		for k, v := range result {
			assert.Regexp(t, regexp.MustCompile(`gcr\.io$`), k, "domains must end with gcr.io")
			assert.Equal(t, "oauth2accesstoken", v.Username)
			assert.Equal(t, "ya29.encryptedtoken", v.Password)
		}
	})

	t.Run("with valid gcloud authentication", func(t *testing.T) {
		cfg := `{"docker": { "authentication" : ["gcloud"]}}`
		viper.Reset()
		viper.SetConfigType("json")
		_ = viper.ReadConfig(strings.NewReader(cfg))

		authMock := &mocks.Authenticator{}
		authMock.On("Authorization").
			Return(&authn.AuthConfig{}, errors.New("test error"))

		auth = authMock

		_, err := GetAuthConfig()
		assert.Error(t, err)
	})

	t.Run("when using the getGcloudAuthString method", func(t *testing.T) {
		cfg := `{"docker": { "authentication" : ["gcloud"]}}`
		viper.Reset()
		viper.SetConfigType("json")
		_ = viper.ReadConfig(strings.NewReader(cfg))

		authMock := &mocks.Authenticator{}
		authMock.On("Authorization").
			Return(&authn.AuthConfig{}, errors.New("test error"))

		auth = authMock

		_, err := getGcloudAuthString()
		assert.Error(t, err)
	})
}

func TestPullImage(t *testing.T) {
	helper := zltest.New(t)
	log.Logger = zerolog.New(helper)
	cfg := `{"docker": { "authentication" : ["gcloud"]}}`
	viper.Reset()
	viper.SetConfigType("json")
	_ = viper.ReadConfig(strings.NewReader(cfg))

	authMock := &mocks.Authenticator{}
	authMock.On("Authorization").
		Return(&authn.AuthConfig{
			Username: "_token",
			Password: "ya29.encryptedtoken",
		}, nil)

	auth = authMock

	t.Run("successful pull", func(t *testing.T) {
		helper.Reset()
		ret := io.NopCloser(strings.NewReader(`{"status": "test line of output"}`))

		b := &mocks.ImageAPIClient{}
		b.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).
			Once().
			Return(ret, nil)
		builder = b

		err := PullImage("busybox")
		assert.NoError(t, err)

		helper.Entries().ExpMsg("\t| test line of output")
	})

	t.Run("malformed return json", func(t *testing.T) {
		helper.Reset()
		ret := io.NopCloser(strings.NewReader(`{"status": "test line of )`))

		b := &mocks.ImageAPIClient{}
		b.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).
			Once().
			Return(ret, nil)
		builder = b

		err := PullImage("busybox")
		assert.NoError(t, err)

		helper.Entries().ExpError("unexpected end of JSON input")
	})
}

func TestPrintOutput(t *testing.T) {
	helper := zltest.New(t)
	log.Logger = zerolog.New(helper)
	t.Run("status output", func(t *testing.T) {
		l := `
			{"status": "test line of status output"}
			{"stream": "test line of stream output"}
			{"status": "test line of error output",  "errorDetail": {"message": "there was\nan error"}}
			{"stream": "test line of\nerror output", "errorDetail": {"message": "there was an error"}}
		`
		err := printOutput(strings.NewReader(l), StatusOutput)
		assert.NoError(t, err)

		helper.Entries().ExpMsg("\t| test line of status output")
		helper.Entries().ExpMsg("\t| there was")
		helper.Entries().ExpMsg("\t| an error")
		helper.Entries().ExpMsg("\t| there was an error")
		helper.Entries().ExpError("unexpected end of JSON input")
	})

	t.Run("stream output", func(t *testing.T) {
		l := `
			{"status": "test line of status output"}
			{"stream": "test line of stream output"}
			{"status": "test line of error output",  "errorDetail": {"message": "there was\nan error"}}
			{"stream": "test line of\nerror output", "errorDetail": {"message": "there was an error"}}
		`
		err := printOutput(strings.NewReader(l), StreamOutput)
		assert.NoError(t, err)

		helper.Entries().ExpMsg("\t| test line of status output")
		helper.Entries().ExpMsg("\t| there was")
		helper.Entries().ExpMsg("\t| an error")
		helper.Entries().ExpMsg("\t| there was an error")
		helper.Entries().ExpError("unexpected end of JSON input")
	})
}
