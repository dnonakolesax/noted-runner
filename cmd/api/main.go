package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/dnonakolesax/noted-runner/internal/configs"
	"github.com/dnonakolesax/noted-runner/internal/consumers"
	compilerDelivery "github.com/dnonakolesax/noted-runner/internal/delivery/compiler/v1/http"
	"github.com/dnonakolesax/noted-runner/internal/docker"
	"github.com/dnonakolesax/noted-runner/internal/logger"
	"github.com/dnonakolesax/noted-runner/internal/rabbit"
	"github.com/dnonakolesax/noted-runner/internal/routing"
	"github.com/dnonakolesax/noted-runner/internal/usecase"

	"github.com/valyala/fasthttp"
)

func main() {
	lcfg := configs.LoggerConfig{LogLevel: "debug", LogAddSource: true}
	initLogger := logger.NewLogger(lcfg, "init")
	router := routing.NewRouter()

	dock, err := docker.NewDockerClient()

	if err != nil {
		initLogger.Error("error creating docker", slog.String("error", err.Error()))
		return
	}

	uc := usecase.NewCompilerUsecase(dock, "/noted/codes/kernels", "noted-kernel_")

	cd := compilerDelivery.NewComilerDelivery(uc)
	router.NewAPIGroup("/compiler", "1", cd)

	
	rmq, err := rabbit.NewRabbit("amqp://guest:guest@rabbit:5672/")

	if err != nil {
		initLogger.Error("error creating rabbit", slog.String("error", err.Error()))
		return
	}

	consumer := consumers.NewRunnerConsumer(rmq.Queue, cd)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)


	srv := fasthttp.Server{
		Handler: router.Router().Handler,
		// Handler: middlewares.CommonMiddleware(router.Router().Handler, appLoggers.HTTP),

		// ReadTimeout:  serverConfig.ReadTimeout,
		// WriteTimeout: serverConfig.WriteTimeout,
		// IdleTimeout:  serverConfig.IdleTimeout,

		// MaxRequestBodySize: serverConfig.MaxReqBodySize,
		// ReadBufferSize:     serverConfig.ReadBufferSize,
		// WriteBufferSize:    serverConfig.WriteBufferSize,

		// Concurrency:        serverConfig.Concurrency,
		// MaxConnsPerIP:      serverConfig.MaxConnsPerIP,
		// MaxRequestsPerConn: serverConfig.MaxRequestsPerConn,

		// TCPKeepalivePeriod: serverConfig.TCPKeepAlivePeriod,
	}

	wg := &sync.WaitGroup{}

	wg.Go(func() {
		initLogger.Info("Starting HTTP server", slog.Int("Port", 8998))
		httpErr := srv.ListenAndServe(":" + strconv.Itoa(8998))
		if httpErr != nil {
			initLogger.Error(fmt.Sprintf("Couldn't start server: %v", httpErr))
		}
	},
	)

	wg.Go(func() {
		consumer.Consume()
	})

	sig := <-quit
	initLogger.Info("Received signal", slog.String("signal", sig.String()))
	err = srv.Shutdown()
	if err != nil {
		initLogger.ErrorContext(context.Background(), "Main HTTP server shutdown error",
			slog.String("error", err.Error()))
	}
	rmq.Close()

	wg.Wait()
}
