package docker

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/dnonakolesax/noted-runner/internal/configs"
	"github.com/dnonakolesax/noted-runner/internal/logger"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type DockerClient struct {
	client *client.Client
	config *configs.DockerConfig
	logger *slog.Logger
}

func NewDockerClient(config *configs.DockerConfig, dckLogger *slog.Logger) (*DockerClient, error) {
	err := os.Setenv("DOCKER_HOST", config.Host)

	if err != nil {
		dckLogger.Error("error setting dockerhost", logger.LogError(err))
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		dckLogger.Error("error creating client", logger.LogError(err))
		return nil, err
	}

	_, err = cli.Ping(context.Background())

	if err != nil {
		dckLogger.Error("error pinging client", logger.LogError(err))
		return nil, err
	}

	return &DockerClient{client: cli, config: config, logger: dckLogger}, err
}

func (dc *DockerClient) Close() {
	_ = dc.client.Close()
}

func (dc *DockerClient) Create(name string, kernelID string) (string, error) {
	ports := make(nat.PortSet)
	ports[nat.Port(dc.config.AppPort)] = struct{}{}

	config := &container.Config{
		Image:        dc.config.Image,
		ExposedPorts: ports,
		Env: []string{"RMQ_ADDR=" + dc.config.Env.RMQAddr, 
					  "KERNEL_ID=" + kernelID, 
					  "MOUNT_PATH=" + dc.config.Env.MountPath, 
					  "EXPORT_PREFIX=" + dc.config.Env.ExportPrefix, 
					  "BLOCK_PREFIX=" + dc.config.Env.BlockPrefix, 
					  "CHAN_NAME=" + dc.config.Env.ChanName, 
					  "BLOCK_TIMEOUT=" + strconv.Itoa(int(dc.config.Env.BlockTimeout.Seconds()))},
	}

	hostConfig := &container.HostConfig{
		//Runtime: "runsc", 
		Mounts: []mount.Mount{{
			Type: mount.TypeVolume,
			Source: dc.config.Volume.Source,
			Target: dc.config.Volume.Target,
		}},
		//AutoRemove: true,
	}

	networkName := dc.config.Network
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {
				NetworkID: networkName,
			},
		},
	}
	resp, err := dc.client.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		networkConfig,
		nil,
		name,
	)
	if err != nil {
		dc.logger.Error("error creating container", logger.LogError(err))
		return "", err
	}
	return resp.ID, nil
}

func (dc *DockerClient) Run(id string) error {
	err := dc.client.ContainerStart(context.Background(), id, container.StartOptions{})
	if err != nil {
		dc.logger.Error("error running container", logger.LogError(err))
	}
	return err
}

func (dc *DockerClient) Remove(id string) error {
	err := dc.client.ContainerStop(context.Background(), id, container.StopOptions{})
	if err != nil {
		dc.logger.Error("error stopping container", logger.LogError(err))
		if client.IsErrConnectionFailed(err) {
			return nil
		}
		return err
	}

	err = dc.client.ContainerRemove(context.Background(), id, container.RemoveOptions{})
	if err != nil {
		dc.logger.Error("error removing container", logger.LogError(err))
		if client.IsErrConnectionFailed(err) {
			return nil
		}
		return err
	}
	return nil
}
