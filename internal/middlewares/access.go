package middlewares

import (
	"context"
	"log/slog"
	"strings"

	"github.com/dnonakolesax/noted-runner/internal/consts"
	access "github.com/dnonakolesax/noted-runner/internal/usecase/access/proto"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/metadata"
)

type AccessMW struct {
	logger *slog.Logger
	client access.AcessServiceClient
}

func NewAccessMW(client access.AcessServiceClient, logger *slog.Logger) *AccessMW {
	return &AccessMW{logger: logger, client: client}
}

func (am *AccessMW) MW(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		trace := string(ctx.Request.Header.Peek(consts.HTTPHeaderXRequestID))
		contex := context.WithValue(context.Background(), consts.TraceContextKey, trace)
		userID := ctx.Request.UserValue(consts.CtxUserIDKey)
		header := metadata.New(map[string]string{"trace_id": string(ctx.Request.Header.Peek("X-Request-Id"))})

		pCtx := metadata.NewOutgoingContext(context.Background(), header)
		access, err := am.client.FileAccessCtx(pCtx, &access.AccessRequest{UserID: userID.(string), FileID: string(ctx.QueryArgs().Peek("kernel-id"))})

		if err != nil {
			am.logger.ErrorContext(contex, "error introspecting", slog.String(consts.ErrorLoggerKey, err.Error()))
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			return
		}

		if !strings.Contains(access.Access, "x") {
			am.logger.WarnContext(contex, "user has no right to execute", slog.String("access", access.Access))
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			return
		}
		h(ctx)
	})
}
