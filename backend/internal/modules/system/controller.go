package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service *Service
}

func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

func (c *Controller) ListConfigs(ctx *gin.Context) {
	items, err := c.service.ListConfigs(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list configs"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) ListSettingsSections(ctx *gin.Context) {
	items, err := c.service.ListSettingsSections(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list settings sections"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) PublicSiteSettings(ctx *gin.Context) {
	item, err := c.service.PublicSiteSettings(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load public site settings"})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) UpsertConfig(ctx *gin.Context) {
	actorID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	var req struct {
		Value map[string]any `json:"value"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	item, err := c.service.UpsertConfig(ctx, actorID, ctx.Param("key"), req.Value)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to upsert config"})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) DeleteConfig(ctx *gin.Context) {
	key := ctx.Param("key")
	if err := c.service.DeleteConfig(ctx, key); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete config"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "deleted", "key": key})
}

func (c *Controller) SendMailDeliveryTest(ctx *gin.Context) {
	actorID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	var req struct {
		To string `json:"to"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	item, err := c.service.SendMailDeliveryTest(ctx, actorID, req.To)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) ListJobs(ctx *gin.Context) {
	items, err := c.service.ListJobs(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list jobs"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) ListAudit(ctx *gin.Context) {
	items, err := c.service.ListAudit(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list audit logs"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func currentUserID(ctx *gin.Context) (uint64, bool) {
	value, exists := ctx.Get("auth.userID")
	if !exists {
		return 0, false
	}
	userID, ok := value.(uint64)
	return userID, ok
}
