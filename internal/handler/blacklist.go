package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neko-server-dev/gomanager/internal/ban"
	"github.com/neko-server-dev/gomanager/internal/errfile"
	"github.com/neko-server-dev/gomanager/internal/nftables"
)

type BlacklistHandler struct {
	service *ban.Service
}

func NewBlacklistHandler(service *ban.Service) *BlacklistHandler {
	return &BlacklistHandler{service: service}
}

type addRequest struct {
	IP        string `json:"ip" binding:"required"`
	TTL       string `json:"ttl,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type listResponse struct {
	Entries []ban.Entry `json:"entries"`
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
	entries, err := h.service.List()
	if err != nil {
		errfile.Record("blacklist list", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	if entries == nil {
		entries = []ban.Entry{}
	}
	c.JSON(http.StatusOK, listResponse{Entries: entries})
}

func (h *BlacklistHandler) add(c *gin.Context) {
	var req addRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "ip is required"})
		return
	}

	expiresAt, err := parseExpiration(req.TTL, req.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if err := h.service.Add(req.IP, expiresAt); err != nil {
		writeBlacklistError(c, err)
		return
	}

	resp := gin.H{"ip": req.IP}
	if expiresAt != nil {
		resp["expires_at"] = expiresAt.Format(time.RFC3339)
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *BlacklistHandler) remove(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "ip is required"})
		return
	}

	if err := h.service.Remove(ip); err != nil {
		writeBlacklistError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func parseExpiration(ttl, expiresAt string) (*time.Time, error) {
	if ttl != "" && expiresAt != "" {
		return nil, errors.New("specify either ttl or expires_at, not both")
	}
	if ttl != "" {
		t, err := ban.ParseTTL(ttl)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}
	if expiresAt != "" {
		t, err := ban.ParseExpiresAt(expiresAt)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}
	return nil, nil
}

func writeBlacklistError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, nftables.ErrInvalidIP):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, nftables.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, ban.ErrInvalidTTL), errors.Is(err, ban.ErrInvalidExpiresAt):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		errfile.Record("blacklist", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
	}
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
