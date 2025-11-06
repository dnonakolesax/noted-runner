package docker

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type DockerClient struct {
	client       *client.Client
	//logger *slog.Logger
}

func NewDockerClient() (*DockerClient, error) {
	err := os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")

	if err != nil {
		slog.Error("error setting dockerhost", slog.String("error", err.Error()))
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("error creating client", slog.String("error", err.Error()))
		return nil, err
	}

	_ , err = cli.Ping(context.Background())

	if err != nil {
		slog.Error("error pinging client", slog.String("error", err.Error()))
		return nil, err
	}

	return &DockerClient{client: cli}, err
}

func (dc *DockerClient) Close() {
	_ = dc.client.Close()
}

func (dc *DockerClient) Create(name string) (string, error) {
	ports := make(nat.PortSet)
	ports["8080"] = struct{}{}
	// Запускаем Go контейнер
	config := &container.Config{
		Image:        "golang:alpine",
		Cmd:          []string{"sh", "-c", "echo 'Hello from nested container!' && go version"},
		ExposedPorts: ports,
	}

	hostConfig := &container.HostConfig{
		//Runtime: "runsc", // Пытаемся использовать gVisor
	}

	networkName := "noted-rmq-runners"
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
		"",
	)
	if err != nil {
		log.Printf("Ошибка создания контейнера (возможно gVisor недоступен): %v", err)
		return "", err
	}
	return resp.ID, nil
}

func (dc *DockerClient) Run (id string) error {
	err := dc.client.ContainerStart(context.Background(), id, container.StartOptions{})
	if err != nil {
		log.Fatalf("Ошибка запуска: %v", err)
	}
	return err
}

func (dc *DockerClient) Remove (id string) error {
	err := dc.client.ContainerRemove(context.Background(), id, container.RemoveOptions{})
	if err != nil {
		log.Fatalf("Ошибка удаления: %v", err)
		if client.IsErrConnectionFailed(err) {
			return nil
		}
		return err
	}
	return nil
}
