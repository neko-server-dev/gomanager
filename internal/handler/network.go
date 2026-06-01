package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neko-server-dev/gomanager/internal/errfile"
	"github.com/neko-server-dev/gomanager/internal/netstats"
)

type NetworkHandler struct {
	collector *netstats.Collector
}

func NewNetworkHandler(collector *netstats.Collector) *NetworkHandler {
	return &NetworkHandler{collector: collector}
}

func (h *NetworkHandler) Register(r gin.IRoutes) {
	r.GET("/network/bandwidth", h.bandwidth)
}

func (h *NetworkHandler) bandwidth(c *gin.Context) {
	usage, err := h.collector.Bandwidth()
	if err != nil {
		writeNetworkError(c, err)
		return
	}
	c.JSON(http.StatusOK, usage)
}

func writeNetworkError(c *gin.Context, err error) {
	if errors.Is(err, netstats.ErrNotSupported) {
		c.JSON(http.StatusNotImplemented, errorResponse{Error: err.Error()})
		return
	}
	errfile.Record("network bandwidth", err)
	c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
}
