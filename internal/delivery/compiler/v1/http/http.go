package http

import (
	"log/slog"

	"github.com/dnonakolesax/noted-runner/internal/logger"
	"github.com/dnonakolesax/noted-runner/internal/model"
	"github.com/fasthttp/router"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type CompilerUsecase interface {
	StartKernel(kernelID string, userID string) (string, error)
	RunBlock(kernelID string, blockID string, userID string) error
	StopKernel(string) error
}

type ComilerDelivery struct {
	activeConns     map[string]*websocket.Conn
	kernelListeners map[string]string
	usecase         CompilerUsecase
	logger          *slog.Logger
}

func NewComilerDelivery(usecase CompilerUsecase, logger *slog.Logger) *ComilerDelivery {
	activeConns := make(map[string]*websocket.Conn)
	kernelListeners := make(map[string]string)
	return &ComilerDelivery{activeConns: activeConns, kernelListeners: kernelListeners, usecase: usecase, logger: logger}
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
		cd.logger.Warn("no kernel id passed")
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	if _, ok := cd.activeConns[userId]; ok {
		cd.logger.Warn("user already connected", slog.String("id", userId))
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	cd.logger.Info("starting kernel", slog.String("id", string(kernelID)))

	id, err := cd.usecase.StartKernel(string(kernelID), userId)
	cd.logger.Info("started kernel", slog.String("container id", id))

	if err != nil {
		cd.logger.Error("error starting kernel", slog.String("error", err.Error()))
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	err = upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		cd.activeConns[userId] = conn
		cd.kernelListeners[string(kernelID)] = userId
		for {
			messageType, message, err := conn.ReadMessage()

			if messageType == websocket.CloseMessage || messageType == -1 {
				delete(cd.activeConns, userId)
				cd.logger.Info("kernelid", slog.String("container id", id))
				_ = cd.usecase.StopKernel(id)
				break
			}

			if err != nil {
				cd.logger.Error("error reading message", logger.LogError(err))
				err := conn.WriteMessage(websocket.TextMessage, []byte("error reading message"))
				if err != nil {
					cd.logger.Error("error sending message", logger.LogError(err))
					err := conn.Close()
					if err != nil {
						cd.logger.Error("error closing conn", logger.LogError(err))
					}
					break
				}
				continue
			}

			cd.logger.Info("received message", slog.String("text", string(message)))
			err = cd.usecase.RunBlock(string(kernelID), string(message), userId)

			if err != nil {
				cd.logger.Error("error compiling", logger.LogError(err))

				resp := model.KernelMessage{}
				resp.KernelID = string(kernelID)
				resp.BlockID = string(message)
				resp.Result = "error compiling:" + err.Error()
				resp.Fail = true

				err := conn.WriteJSON(resp)

				if err != nil {
					cd.logger.Error("error sending message", logger.LogError(err))
					err := conn.Close()
					if err != nil {
						cd.logger.Error("error closing conn",logger.LogError(err))
					}
					break
				}
			}
		}
	})
	if err != nil {
		cd.logger.Error("error upgrading", logger.LogError(err))
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
}

func (cd *ComilerDelivery) SendMemes(kernelId string, memes string) {
	userId := cd.kernelListeners[kernelId]
	if conn, ok := cd.activeConns[userId]; ok {
		err := conn.WriteMessage(websocket.TextMessage, []byte(memes))
		if err != nil {
			cd.logger.Error("error sending message", logger.LogError(err))
		}
	} else {
		cd.logger.Error("couldn't find user", slog.String("id", userId))
	}
}

func (cd *ComilerDelivery) RegisterRoutes(apiGroup *router.Group) {
	group := apiGroup.Group("/ws")
	group.ANY("/", cd.Compile)
}
