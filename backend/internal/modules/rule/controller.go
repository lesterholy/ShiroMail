package rule

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

func (c *Controller) List(ctx *gin.Context) {
	items, err := c.service.List(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list rules"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) Upsert(ctx *gin.Context) {
	actorID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	var req struct {
		Name           string `json:"name"`
		RetentionHours int    `json:"retentionHours"`
		AutoExtend     bool   `json:"autoExtend"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	item, err := c.service.Upsert(ctx, actorID, Rule{
		ID:             ctx.Param("id"),
		Name:           req.Name,
		RetentionHours: req.RetentionHours,
		AutoExtend:     req.AutoExtend,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to upsert rule"})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func currentUserID(ctx *gin.Context) (uint64, bool) {
	value, exists := ctx.Get("auth.userID")
	if !exists {
		return 0, false
	}
	userID, ok := value.(uint64)
	return userID, ok
}
