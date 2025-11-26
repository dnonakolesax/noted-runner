package docker

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/dnonakolesax/noted-runner/internal/configs"
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

func NewDockerClient(config *configs.DockerConfig, logger *slog.Logger) (*DockerClient, error) {
	err := os.Setenv("DOCKER_HOST", config.Host)

	if err != nil {
		slog.Error("error setting dockerhost", slog.String("error", err.Error()))
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("error creating client", slog.String("error", err.Error()))
		return nil, err
	}

	_, err = cli.Ping(context.Background())

	if err != nil {
		slog.Error("error pinging client", slog.String("error", err.Error()))
		return nil, err
	}

	return &DockerClient{client: cli, config: config, logger: logger}, err
}

func (dc *DockerClient) Close() {
	_ = dc.client.Close()
}

func (dc *DockerClient) Create(name string, kernelID string) (string, error) {
	ports := make(nat.PortSet)
	ports[nat.Port(dc.config.AppPort)] = struct{}{}
	// Запускаем Go контейнер
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
		//Runtime: "runsc", // Пытаемся использовать gVisor
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
		log.Printf("Ошибка создания контейнера (возможно gVisor недоступен): %v", err)
		return "", err
	}
	return resp.ID, nil
}

func (dc *DockerClient) Run(id string) error {
	err := dc.client.ContainerStart(context.Background(), id, container.StartOptions{})
	if err != nil {
		log.Fatalf("Ошибка запуска: %v", err)
	}
	return err
}

func (dc *DockerClient) Remove(id string) error {
	err := dc.client.ContainerStop(context.Background(), id, container.StopOptions{})
	if err != nil {
		log.Fatalf("Ошибка остановки: %v", err)
		if client.IsErrConnectionFailed(err) {
			return nil
		}
		return err
	}

	err = dc.client.ContainerRemove(context.Background(), id, container.RemoveOptions{})
	if err != nil {
		log.Fatalf("Ошибка удаления: %v", err)
		if client.IsErrConnectionFailed(err) {
			return nil
		}
		return err
	}
	return nil
}
