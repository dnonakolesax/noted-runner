package application

import (
	"github.com/dnonakolesax/noted-runner/internal/consumers"
	compilerDelivery "github.com/dnonakolesax/noted-runner/internal/delivery/compiler/v1/http"
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
	uc := usecase.NewCompilerUsecase(a.components.Docker, a.configs.Docker.Env.MountPath, a.configs.Docker.Prefix)

	/************************************************/
	/*                DELIVERY INIT                 */
	/************************************************/
	cd := compilerDelivery.NewComilerDelivery(uc)
	a.layers.compileHTTP = cd

	/************************************************/
	/*                CONSUMERS INIT                */
	/************************************************/
	consumer := consumers.NewRunnerConsumer(a.components.Rabbit.Queue, cd)
	a.layers.compileResultConsumer = consumer
	return nil
}
