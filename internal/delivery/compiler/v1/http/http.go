package http

import (
	"log/slog"

	"github.com/fasthttp/router"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type CompilerUsecase interface {
	StartKernel(string) (string, error)
	RunBlock(kernelID string, blockID string) error
	StopKernel(string) error
}

type ComilerDelivery struct {
	activeConns map[string]*websocket.Conn
	usecase     CompilerUsecase
}

func NewComilerDelivery(usecase CompilerUsecase) *ComilerDelivery {
	activeConns := make(map[string]*websocket.Conn)
	return &ComilerDelivery{activeConns: activeConns, usecase: usecase}
}

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(ctx *fasthttp.RequestCtx) bool { return true },
}

func (cd *ComilerDelivery) Compile(ctx *fasthttp.RequestCtx) {
	userId := "1"
	kernelID := ctx.QueryArgs().Peek("kernel-id")

	if kernelID == nil {
		slog.Warn("no kernel id passed")
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	if _, ok := cd.activeConns[userId]; ok {
		slog.Warn("user already connected", slog.String("id", userId))
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	// check access

	slog.Info("starting kernel", slog.String("id", string(kernelID)))

	id, err := cd.usecase.StartKernel(string(kernelID))
	slog.Info("started kernel", slog.String("container id", id))

	if err != nil {
		slog.Error("error starting kernel", slog.String("error", err.Error()))
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	err = upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		cd.activeConns[userId] = conn
		for {
			messageType, message, err := conn.ReadMessage()

			if messageType == websocket.CloseMessage || messageType == -1 {
				delete(cd.activeConns, userId)
				_ = cd.usecase.StopKernel(string(kernelID))
				break
			}

			if err != nil {
				slog.Error("error reading message", slog.String("error", err.Error()))
				err := conn.WriteMessage(websocket.TextMessage, []byte("error reading message"))
				if err != nil {
					slog.Error("error sending message", slog.String("error", err.Error()))
					err := conn.Close()
					if err != nil {
						slog.Error("error closing conn", slog.String("error", err.Error()))
						break
					}
				}
				continue
			}

			slog.Info("received message", slog.String("text", string(message)))
			err = cd.usecase.RunBlock(string(kernelID), string(message))

			if err != nil {
				slog.Error("error compiling", slog.String("error", err.Error()))
				_ = conn.WriteMessage(websocket.TextMessage, []byte("error compiling:" + err.Error()))
			}
		}
	})
	if err != nil {
		slog.Error("error upgrading", slog.String("error", err.Error()))
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
}

func (cd *ComilerDelivery) SendMemes(userId string, memes string) {
	if conn, ok := cd.activeConns[userId]; ok {
		err := conn.WriteMessage(websocket.TextMessage, []byte(memes))
		if err != nil {
			slog.Error("error sending message", slog.String("error", err.Error()))
		}
	} else {
		slog.Error("couldn't find user", slog.String("id", userId))
	}
}

func (cd *ComilerDelivery) RegisterRoutes(apiGroup *router.Group) {
	group := apiGroup.Group("/ws")
	group.ANY("/", cd.Compile)
}
