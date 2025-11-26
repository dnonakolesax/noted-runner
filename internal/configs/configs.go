package configs

import (
	"context"
	"log/slog"

	"github.com/dnonakolesax/noted-runner/internal/consts"
	"github.com/dnonakolesax/viper"
	"github.com/joho/godotenv"
)

type Config struct {
	Docker *DockerConfig

	HTTPClient *HTTPClientConfig
	HTTPServer *HTTPServerConfig

	Service *ServiceConfig
	Logger  *LoggerConfig
}

func SetupConfigs(initLogger *slog.Logger, configsDir string) (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		initLogger.ErrorContext(context.Background(), "Error loading .env file")
		return nil, err
	}

	v := viper.New()
	v.PanicOnNil = true

	appConfig := &ServiceConfig{}
	serverConfig := &HTTPServerConfig{}
	httpClientConfig := &HTTPClientConfig{}
	loggerConfig := &LoggerConfig{}
	dockerConfig := &DockerConfig{}

	err = Load(configsDir, v, initLogger, appConfig, serverConfig, httpClientConfig, loggerConfig, dockerConfig)

	if err != nil {
		initLogger.ErrorContext(context.Background(), "Error loading config",
			slog.String(consts.ErrorLoggerKey, err.Error()))
		return nil, err
	}

	
	return &Config{
		Docker: dockerConfig,

		HTTPClient: httpClientConfig,
		HTTPServer: serverConfig,

		Service: appConfig,
		Logger: loggerConfig,
	}, nil
}
