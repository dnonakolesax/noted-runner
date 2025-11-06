package configs

import "time"

type LoggerConfig struct {
	LogLevel       string
	LogTimeout     time.Duration
	LogAddSource   bool
	LogMaxFileSize int
	LogMaxBackups  int
	LogMaxAge      int
}