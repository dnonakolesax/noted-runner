package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	fasthttpprom "github.com/carousell/fasthttp-prometheus-middleware"
	"github.com/valyala/fasthttp"

	"github.com/dnonakolesax/noted-runner/internal/configs"
	"github.com/dnonakolesax/noted-runner/internal/logger"
	"github.com/dnonakolesax/noted-runner/internal/middlewares"
	"github.com/dnonakolesax/noted-runner/internal/routing"
)

type App struct {
	configs    *configs.Config
	health     *HealthChecks
	layers     *Layers
	metrics    *Metrics
	initLogger *slog.Logger
	loggers    *logger.Loggers
	components *Components
}

func NewApp(configsDir string) (*App, error) {
	lcfg := &configs.LoggerConfig{LogLevel: "info", LogAddSource: true}
	initLogger := logger.NewLogger(lcfg, "init")
	app := &App{}

	app.initLogger = initLogger

	app.InitHealthchecks()

	configs, err := configs.SetupConfigs(initLogger, configsDir)

	if err != nil {
		return nil, err
	}

	app.configs = configs

	loggers := logger.SetupLoggers(app.configs.Logger)

	app.loggers = loggers

	app.SetupMetrics()

	err = app.SetupComponents()

	if err != nil {
		return nil, err
	}

	err = app.SetupLayers()

	if err != nil {
		return nil, err
	}

	return app, nil
}

func (a *App) Run() {
	/************************************************/
	/*               HTTP ROUTER SETUP              */
	/************************************************/

	router := routing.NewRouter()
	p := fasthttpprom.NewPrometheus("")
	p.Use(router.Router())
	router.NewAPIGroup(a.configs.Service.BasePath, "1",
		a.layers.compileHTTP)

	wg := &sync.WaitGroup{}

	/************************************************/
	/*               HTTP SERVER START              */
	/************************************************/

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	srv := fasthttp.Server{
		Handler: middlewares.CommonMiddleware(router.Router().Handler, a.loggers.HTTP),

		ReadTimeout:  a.configs.HTTPServer.ReadTimeout,
		WriteTimeout: a.configs.HTTPServer.WriteTimeout,
		IdleTimeout:  a.configs.HTTPServer.IdleTimeout,

		MaxRequestBodySize: a.configs.HTTPServer.MaxReqBodySize,
		ReadBufferSize:     1024*1024,
		WriteBufferSize:    a.configs.HTTPServer.WriteBufferSize,

		Concurrency:        a.configs.HTTPServer.Concurrency,
		MaxConnsPerIP:      a.configs.HTTPServer.MaxConnsPerIP,
		MaxRequestsPerConn: a.configs.HTTPServer.MaxRequestsPerConn,

		TCPKeepalivePeriod: a.configs.HTTPServer.TCPKeepAlivePeriod,
	}

	wg.Go(func() {
		a.initLogger.Info("Starting HTTP server", slog.Int("Port", a.configs.Service.Port))
		httpErr := srv.ListenAndServe(":" + strconv.Itoa(a.configs.Service.Port))
		if httpErr != nil {
			a.initLogger.Error(fmt.Sprintf("Couldn't start server: %v", httpErr))
		}
	},
	)

	/************************************************/
	/*              RMQ CONSUMER START              */
	/************************************************/
	wg.Go(func() {
		a.layers.compileResultConsumer.Consume()
	})

	/************************************************/
	/*              SHUTDOWN SIGNAL RCV             */
	/************************************************/
	sig := <-quit
	a.initLogger.Info("Received signal", slog.String("signal", sig.String()))

	err := srv.Shutdown()
	if err != nil {
		a.initLogger.ErrorContext(context.Background(), "Main HTTP server shutdown error",
			slog.String("error", err.Error()))
	}

	a.components.Rabbit.Close()

	wg.Wait()
}
