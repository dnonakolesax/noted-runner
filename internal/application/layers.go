package application

import (
	"github.com/dnonakolesax/noted-runner/internal/consumers"
	compilerDelivery "github.com/dnonakolesax/noted-runner/internal/delivery/compiler/v1/http"
	"github.com/dnonakolesax/noted-runner/internal/middlewares"
	"github.com/dnonakolesax/noted-runner/internal/usecase"
)

type Layers struct {
	compileHTTP *compilerDelivery.ComilerDelivery

	compileResultConsumer *consumers.RunnerConsumer
}

func (a *App) SetupLayers() error {
	a.layers = &Layers{}
	/************************************************/
	/*                USECASES INIT                 */
	/************************************************/
	uc := usecase.NewCompilerUsecase(a.components.Docker, a.configs.Docker.Env.MountPath, a.configs.Docker.Prefix,
		a.loggers.Service, a.configs.Service, a.components.HTTPC)

	/************************************************/
	/*              MIDDLEWARE INIT                 */
	/************************************************/
	authMW := middlewares.NewAuthMW(*a.components.GRPCAC, a.loggers.HTTP)
	accessMW := middlewares.NewAccessMW(*a.components.GRPCAcC, a.loggers.HTTP)

	/************************************************/
	/*                DELIVERY INIT                 */
	/************************************************/
	cd := compilerDelivery.NewComilerDelivery(uc, a.loggers.HTTP, authMW, accessMW)
	a.layers.compileHTTP = cd

	/************************************************/
	/*                CONSUMERS INIT                */
	/************************************************/
	consumer := consumers.NewRunnerConsumer(a.components.Rabbit.Queue, cd, a.loggers.Infra)
	a.layers.compileResultConsumer = consumer
	return nil
}
