package application

import (
	"context"
	"log/slog"

	"github.com/dnonakolesax/noted-runner/internal/consts"
	"github.com/dnonakolesax/noted-runner/internal/docker"
	"github.com/dnonakolesax/noted-runner/internal/httpclient"
	"github.com/dnonakolesax/noted-runner/internal/rabbit"
)

type Components struct {
	Docker *docker.DockerClient
	Rabbit *rabbit.RabbitQueue
	HTTPC  *httpclient.HTTPClient
}

func (a *App) SetupComponents() error {
	a.components = &Components{}
	/************************************************/
	/*               RABBIT CONNECTION              */
	/************************************************/
	a.initLogger.InfoContext(context.Background(), "Starting RabbitMQ connection")

	rmq, err := rabbit.NewRabbit(a.configs.Docker.Env.RMQAddr, a.configs.Docker.Env.ChanName, a.loggers.Infra)

	if err != nil {
		a.initLogger.ErrorContext(context.Background(), "Error connecting to RabbitMQ",
			slog.String(consts.ErrorLoggerKey, err.Error()))
		return err
	}
	a.initLogger.InfoContext(context.Background(), "RabbitMQ connection established")
	a.components.Rabbit = rmq

	/************************************************/
	/*              DOCKER CLIENT INIT              */
	/************************************************/

	a.initLogger.InfoContext(context.Background(), "Creating docker client")
	dock, err := docker.NewDockerClient(a.configs.Docker, a.loggers.Infra)

	if err != nil {
		a.initLogger.ErrorContext(context.Background(), "Error creating docker client",
			slog.String(consts.ErrorLoggerKey, err.Error()))
		return err
	}
	a.initLogger.InfoContext(context.Background(), "Docker client created")
	a.components.Docker = dock

	
	/************************************************/
	/*               HTTP CLIENT INIT               */
	/************************************************/
	a.initLogger.InfoContext(context.Background(), "Creating HTTP client")
	httpc, err := httpclient.NewWithRetry(a.configs.HTTPClient, a.metrics.RunnerMetrics, a.loggers.HTTPc)
	if err != nil {
		a.initLogger.ErrorContext(context.Background(), "Error creating http client",
			slog.String(consts.ErrorLoggerKey, err.Error()))
		return err
	}
	a.initLogger.InfoContext(context.Background(), "HTTP client created")
	a.components.HTTPC = httpc

	return nil
}
