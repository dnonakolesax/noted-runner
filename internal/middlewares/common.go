package middlewares

import (
	"encoding/base64"
	"log/slog"

	"github.com/valyala/fasthttp"

	"github.com/dnonakolesax/noted-runner/internal/rnd"
)

const requestIDSize = 16

func CommonMiddleware(h fasthttp.RequestHandler, logger *slog.Logger) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		requestID := ctx.Request.Header.Peek("X-Request-Id")
		var reqID string
		if requestID == nil {
			requestID = rnd.NotSafeGenRandomString(requestIDSize)
			reqID = base64.RawURLEncoding.EncodeToString(requestID)
			ctx.Request.Header.Set("X-Request-Id", reqID)
		} else {
			reqID = string(requestID)
		}
		logger.Info("Received Request",
			slog.String("method", string(ctx.Method())),
			slog.String("path", string(ctx.Path())),
			slog.String("ip", ctx.RemoteIP().String()),
			slog.String("requestId", reqID),
			slog.String("userAgent", string(ctx.UserAgent())),
		)
		now := ctx.Time().UnixMilli()
		h(ctx)
		end := ctx.Time().UnixMilli()
		logger.Info("Completed request",
			slog.String("method", string(ctx.Method())),
			slog.String("path", string(ctx.Path())),
			slog.String("requestId", reqID),
			slog.Int("status", ctx.Response.StatusCode()),
			slog.Int("duration", int(end-now)),
		)
	})
}
