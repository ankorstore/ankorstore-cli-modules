package docker

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/go-errors/errors"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	ErrCannotRedeclare = errors.New("cannot be declared more than once")
)

type RunOpt func(*RunConfig) error

type RunCommand interface {
	GetCommand() ([]string, error)
}

type RunConfig struct {
	Config        *container.Config
	HostConfig    *container.HostConfig
	NetworkConfig *network.NetworkingConfig
	Platform      *specs.Platform
	Name          string
}

func initRunConfig(cfg *RunConfig) {
	if cfg.Config == nil {
		cfg.Config = &container.Config{}
	}
}

func initRunHostConfig(cfg *RunConfig) {
	if cfg.HostConfig == nil {
		cfg.HostConfig = &container.HostConfig{}
	}
}

func RunWithImage(image string) RunOpt {
	return func(cfg *RunConfig) error {
		initRunConfig(cfg)
		cfg.Config.Image = image
		return nil
	}
}

func RunWithWorkingDir(dir string) RunOpt {
	return func(cfg *RunConfig) error {
		initRunConfig(cfg)
		cfg.Config.WorkingDir = dir
		return nil
	}
}

func RunWithEntrypoint(entrypoint []string) RunOpt {
	return func(cfg *RunConfig) error {
		initRunConfig(cfg)
		cfg.Config.Entrypoint = entrypoint
		return nil
	}
}

func RunWithCommand(cmd []string) RunOpt {
	return func(cfg *RunConfig) error {
		initRunConfig(cfg)
		cfg.Config.Cmd = cmd
		return nil
	}
}

func RunWithMounts(mounts []mount.Mount) RunOpt {
	return func(cfg *RunConfig) error {
		initRunHostConfig(cfg)
		if cfg.Name != "" {
			return errors.New(fmt.Errorf("'Mounts' %w", ErrCannotRedeclare))
		}
		cfg.HostConfig.Mounts = mounts
		return nil
	}
}

func RunWithShell(shell []string) RunOpt {
	return func(cfg *RunConfig) error {
		initRunConfig(cfg)
		if cfg.Name != "" {
			return errors.New(fmt.Errorf("'Mounts' %w", ErrCannotRedeclare))
		}
		cfg.Config.Shell = shell
		return nil
	}
}
