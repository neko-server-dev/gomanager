package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neko-server-dev/goban/internal/errfile"
	"github.com/neko-server-dev/goban/internal/nftables"
)

type BlacklistHandler struct {
	manager *nftables.Manager
}

func NewBlacklistHandler(manager *nftables.Manager) *BlacklistHandler {
	return &BlacklistHandler{manager: manager}
}

type addRequest struct {
	IP string `json:"ip" binding:"required"`
}

type listResponse struct {
	IPs []string `json:"ips"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *BlacklistHandler) Register(r gin.IRoutes) {
	r.GET("/blacklist", h.list)
	r.POST("/blacklist", h.add)
	r.DELETE("/blacklist/:ip", h.remove)
}

func (h *BlacklistHandler) list(c *gin.Context) {
	ips, err := h.manager.List()
	if err != nil {
		errfile.Record("blacklist list", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	if ips == nil {
		ips = []string{}
	}
	c.JSON(http.StatusOK, listResponse{IPs: ips})
}

func (h *BlacklistHandler) add(c *gin.Context) {
	var req addRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "ip is required"})
		return
	}

	if err := h.manager.Add(req.IP); err != nil {
		writeBlacklistError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ip": req.IP})
}

func (h *BlacklistHandler) remove(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "ip is required"})
		return
	}

	if err := h.manager.Remove(ip); err != nil {
		writeBlacklistError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func writeBlacklistError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, nftables.ErrInvalidIP):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, nftables.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		errfile.Record("blacklist", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
	}
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
