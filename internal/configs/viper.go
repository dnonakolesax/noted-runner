package configs

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dnonakolesax/viper"

	"github.com/dnonakolesax/noted-runner/internal/consts"
)

type configurable interface {
	SetDefaults(v *viper.Viper)
	Load(v *viper.Viper)
}

func Load(path string, v *viper.Viper, logger *slog.Logger, configs ...configurable) error {
	for _, cfg := range configs {
		cfg.SetDefaults(v)
	}

	v.AddConfigPath(path)
	v.AddConfigPath("./")

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	err := v.MergeInConfig()

	if err != nil {
		var vErr viper.ConfigFileNotFoundError
		if errors.As(err, &vErr) {
			logger.Error("Config file not found yaml")
			return nil
		}
		logger.Error("Failed to merge yaml config", slog.String(consts.ErrorLoggerKey, err.Error()))
		return fmt.Errorf("failed to merge config: %w", err)
	}

	v.SetConfigFile(".env")
	v.SetConfigType("env")
	err = v.MergeInConfig()

	if err != nil {
		var vErr viper.ConfigFileNotFoundError
		if errors.As(err, &vErr) {
			logger.Error("Config file not found env")
			return nil
		}
		logger.Error("Failed to merge dotenv config", slog.String(consts.ErrorLoggerKey, err.Error()))
		return fmt.Errorf("failed to merge config: %w", err)
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	for _, cfg := range configs {
		cfg.Load(v)
	}

	return nil
}
