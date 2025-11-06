package routing

import (
	"github.com/fasthttp/router"
)

type HTTPHandler interface {
	RegisterRoutes(apiGroup *router.Group)
}

type Router struct {
	rtr *router.Router
}

func NewRouter() *Router {
	rtr := router.New()

	return &Router{
		rtr: rtr,
	}
}

func (rr *Router) NewAPIGroup(basePath string, version string, handlers ...HTTPHandler) {
	apiGroup := rr.rtr.Group("/api/v" + version + basePath)

	for _, handler := range handlers {
		handler.RegisterRoutes(apiGroup)
	}
}

func (rr *Router) Router() *router.Router {
	return rr.rtr
}
