package logger

import (
	"log/slog"

	"github.com/dnonakolesax/noted-runner/internal/consts"
)

func LogError(err error) slog.Attr {
	return slog.String(consts.ErrorLoggerKey, err.Error())
}
