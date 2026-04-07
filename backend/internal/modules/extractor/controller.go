package extractor

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"shiro-email/backend/internal/modules/mailbox"
	"shiro-email/backend/internal/modules/message"
)

type Controller struct {
	service *Service
}

func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

func (c *Controller) ListPortalRules(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	items, err := c.service.ListPortalRules(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list extractor rules"})
		return
	}
	ctx.JSON(http.StatusOK, items)
}

func (c *Controller) CreatePortalRule(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req UpsertRuleInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	item, err := c.service.CreatePortalRule(ctx, userID, req)
	if err != nil {
		c.writeError(ctx, err, "failed to create extractor rule")
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

func (c *Controller) UpdatePortalRule(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	ruleID, ok := parseRuleID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	var req UpsertRuleInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	item, err := c.service.UpdatePortalRule(ctx, userID, ruleID, req)
	if err != nil {
		c.writeError(ctx, err, "failed to update extractor rule")
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) DeletePortalRule(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	ruleID, ok := parseRuleID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	if err := c.service.DeletePortalRule(ctx, userID, ruleID); err != nil {
		c.writeError(ctx, err, "failed to delete extractor rule")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (c *Controller) TestPortalRule(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req struct {
		Rule   UpsertRuleInput `json:"rule"`
		Sample RuleTestInput   `json:"sample"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	item, err := c.service.TestPortalRule(ctx, userID, req.Rule, req.Sample)
	if err != nil {
		c.writeError(ctx, err, "failed to test extractor rule")
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) EnableTemplate(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	ruleID, ok := parseRuleID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	if err := c.service.EnableTemplate(ctx, userID, ruleID); err != nil {
		c.writeError(ctx, err, "failed to enable template")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (c *Controller) DisableTemplate(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	ruleID, ok := parseRuleID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	if err := c.service.DisableTemplate(ctx, userID, ruleID); err != nil {
		c.writeError(ctx, err, "failed to disable template")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (c *Controller) CopyTemplate(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	ruleID, ok := parseRuleID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	item, err := c.service.CopyTemplateToUser(ctx, userID, ruleID)
	if err != nil {
		c.writeError(ctx, err, "failed to copy template")
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

func (c *Controller) ListMessageExtractions(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	mailboxID, messageID, ok := parseMailboxMessageIDs(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid mailbox or message id"})
		return
	}
	item, err := c.service.ExtractForPortalMessage(ctx, userID, mailboxID, messageID)
	if err != nil {
		c.writeError(ctx, err, "failed to extract message content")
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) ListAdminRules(ctx *gin.Context) {
	items, err := c.service.ListAdminRules(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list extractor templates"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) CreateAdminRule(ctx *gin.Context) {
	actorID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req UpsertRuleInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	item, err := c.service.CreateAdminRule(ctx, actorID, req)
	if err != nil {
		c.writeError(ctx, err, "failed to create extractor template")
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

func (c *Controller) UpdateAdminRule(ctx *gin.Context) {
	actorID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	ruleID, ok := parseRuleID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	var req UpsertRuleInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	item, err := c.service.UpdateAdminRule(ctx, actorID, ruleID, req)
	if err != nil {
		c.writeError(ctx, err, "failed to update extractor template")
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) DeleteAdminRule(ctx *gin.Context) {
	actorID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	ruleID, ok := parseRuleID(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	if err := c.service.DeleteAdminRule(ctx, actorID, ruleID); err != nil {
		c.writeError(ctx, err, "failed to delete extractor template")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (c *Controller) TestAdminRule(ctx *gin.Context) {
	var req struct {
		Rule   UpsertRuleInput `json:"rule"`
		Sample RuleTestInput   `json:"sample"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	item, err := c.service.TestAdminRule(ctx, req.Rule, req.Sample)
	if err != nil {
		c.writeError(ctx, err, "failed to test extractor template")
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) ListAdminMessageExtractions(ctx *gin.Context) {
	mailboxID, messageID, ok := parseMailboxMessageIDs(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid mailbox or message id"})
		return
	}
	item, err := c.service.ExtractForAdminMessage(ctx, mailboxID, messageID)
	if err != nil {
		c.writeError(ctx, err, "failed to extract message content")
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) writeError(ctx *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, ErrRuleNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"message": "extractor rule not found"})
	case errors.Is(err, ErrInvalidPattern):
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid regex pattern"})
	case errors.Is(err, ErrInvalidRule):
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid extractor rule"})
	case errors.Is(err, mailbox.ErrMailboxNotFound), errors.Is(err, message.ErrMessageDeleted), errors.Is(err, message.ErrAttachmentNotFound), errors.Is(err, message.ErrMessageContentUnavailable):
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": fallback})
	}
}

func currentUserID(ctx *gin.Context) (uint64, bool) {
	value, exists := ctx.Get("auth.userID")
	if !exists {
		return 0, false
	}
	userID, ok := value.(uint64)
	return userID, ok
}

func parseRuleID(ctx *gin.Context) (uint64, bool) {
	value, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseMailboxMessageIDs(ctx *gin.Context) (uint64, uint64, bool) {
	mailboxID, err := strconv.ParseUint(ctx.Param("mailboxId"), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	messageID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return mailboxID, messageID, true
}
