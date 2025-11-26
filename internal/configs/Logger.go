package configs

import (
	"time"

	"github.com/dnonakolesax/viper"
)
const (
	logLevelKey           = "service.log-level"
	logLevelDefault       = "info"
	logAddSourceKey       = "service.log-add-source"
	logAddSourceDefault   = true
	logTimeoutKey         = "service.log-timeout"
	logTimeoutDefault     = 10 * time.Second
	logMaxFileSizeKey     = "service.log-max-file-size"
	logMaxFileSizeDefault = 100
	logMaxBackupsKey      = "service.log-max-backups"
	logMaxBackupsDefault  = 3
	logMaxAgeKey          = "service.log-max-age"
	logMaxAgeDefault      = 28
)

type LoggerConfig struct {
	LogLevel       string
	LogTimeout     time.Duration
	LogAddSource   bool
	LogMaxFileSize int
	LogMaxBackups  int
	LogMaxAge      int
}

func (lc *LoggerConfig) SetDefaults(v *viper.Viper) {
	v.SetDefault(logLevelKey, logLevelDefault)
	v.SetDefault(logAddSourceKey, logAddSourceDefault)
	v.SetDefault(logTimeoutKey, logTimeoutDefault)
	v.SetDefault(logMaxFileSizeKey, logMaxFileSizeDefault)
	v.SetDefault(logMaxBackupsKey, logMaxBackupsDefault)
	v.SetDefault(logMaxAgeKey, logMaxAgeDefault)
}

func (lc *LoggerConfig) Load(v *viper.Viper) {
	lc.LogLevel = v.GetString(logLevelKey)
	lc.LogAddSource = v.GetBool(logAddSourceKey)
	lc.LogTimeout = v.GetDuration(logTimeoutKey)
	lc.LogMaxFileSize = v.GetInt(logMaxFileSizeKey)
	lc.LogMaxBackups = v.GetInt(logMaxBackupsKey)
	lc.LogMaxAge = v.GetInt(logMaxAgeKey)
}
