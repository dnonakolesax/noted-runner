package middlewares

import (
	"context"
	"log/slog"

	"github.com/dnonakolesax/noted-runner/internal/consts"
	"github.com/dnonakolesax/noted-runner/internal/cookies"
	auth "github.com/dnonakolesax/noted-runner/internal/usecase/auth/proto"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/metadata"
)

type AuthMW struct {
	logger *slog.Logger
	client auth.AuthServiceClient
}

func NewAuthMW(client auth.AuthServiceClient, logger *slog.Logger) *AuthMW {
	return &AuthMW{logger: logger, client: client}
}

func (am *AuthMW) AuthMiddleware(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		trace := string(ctx.Request.Header.Peek(consts.HTTPHeaderXRequestID))
		contex := context.WithValue(context.Background(), consts.TraceContextKey, trace)
		at := ctx.Request.Header.Cookie(consts.ATCookieKey)
		if at == nil {
			am.logger.WarnContext(contex, "no at passed")
		}
		rt := ctx.Request.Header.Cookie(consts.RTCookieKey)
		if rt == nil {
			am.logger.WarnContext(contex, "no rt passed")
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			return
		}
		header := metadata.New(map[string]string{"trace_id": "12345"})

		pCtx := metadata.NewOutgoingContext(context.Background(), header)
		tokens, err := am.client.AuthUserIDCtx(pCtx, &auth.UserTokens{Auth: string(at), Refresh: string(rt)})

		if err != nil {
			am.logger.ErrorContext(contex, "error introspecting", slog.String(consts.ErrorLoggerKey, err.Error()))
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			return
		}
		ctx.Request.SetUserValue(consts.CtxUserIDKey, tokens.ID)
		if tokens.At != nil && tokens.Rt != nil&& tokens.It != nil {
			cookies.SetupAccessCookies(ctx, *tokens.At, *tokens.Rt, *tokens.It)
		}
		// am.logger.Debug(dto.AccessToken)
		// am.logger.Debug(dto.RefreshToken)
		// am.logger.Debug(dto.IDToken)
		h(ctx)
	})
}
