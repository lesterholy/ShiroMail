package domain

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"shiro-email/backend/internal/middleware"
	"shiro-email/backend/internal/modules/portal"
)

type Controller struct {
	service *Service
}

func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

func (c *Controller) List(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	items, err := c.service.ListAccessibleActive(ctx, userID, currentAPIKeyArgs(ctx)...)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list domains"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) Create(ctx *gin.Context) {
	var req CreateDomainRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	item, err := c.service.CreateOwned(ctx, userID, req, currentAPIKeyArgs(ctx)...)
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrAPIKeyForbidden):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
		case errors.Is(err, ErrInvalidDomain):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		case errors.Is(err, ErrDomainAlreadyExists):
			ctx.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create domain"})
		}
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

func (c *Controller) Generate(ctx *gin.Context) {
	var req GenerateSubdomainsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	items, err := c.service.GenerateOwnedSubdomains(ctx, userID, req, currentAPIKeyArgs(ctx)...)
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrAPIKeyForbidden):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
		case errors.Is(err, ErrInvalidDomain):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		case errors.Is(err, ErrDomainNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate subdomains"})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) Delete(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	domainID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid domain id"})
		return
	}

	if err := c.service.DeleteOwned(ctx, userID, domainID); err != nil {
		switch {
		case errors.Is(err, ErrDomainNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrDomainHasChildren), errors.Is(err, ErrDomainHasMailboxes):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete domain"})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (c *Controller) UpdateOwnedProviderBinding(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	domainID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid domain id"})
		return
	}

	var req UpdateOwnedDomainProviderBindingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	item, err := c.service.UpdateOwnedDomainProviderBinding(ctx, userID, domainID, req, currentAPIKeyArgs(ctx)...)
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrAPIKeyForbidden):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
		case errors.Is(err, ErrDomainNotFound), errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update domain provider binding"})
		}
		return
	}

	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) VerifyOwnedDomain(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	domainID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid domain id"})
		return
	}

	result, err := c.service.VerifyOwnedDomain(ctx, userID, domainID, currentAPIKeyArgs(ctx)...)
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrAPIKeyForbidden):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
		case errors.Is(err, ErrDomainNotFound), errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAdapterUnavailable), errors.Is(err, ErrInvalidDNSChangeSetRequest):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) GenerateAdmin(ctx *gin.Context) {
	var req GenerateSubdomainsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	items, err := c.service.GenerateSubdomains(ctx, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidDomain):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		case errors.Is(err, ErrDomainNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate subdomains"})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) RequestPublicPoolPublication(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	domainID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid domain id"})
		return
	}

	item, err := c.service.RequestPublicPoolPublication(ctx, userID, domainID, currentAPIKeyArgs(ctx)...)
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrAPIKeyForbidden):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
		case errors.Is(err, ErrDomainNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrInvalidPublicationState):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to request public pool publication"})
		}
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) WithdrawPublicPoolPublication(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	domainID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid domain id"})
		return
	}

	item, err := c.service.WithdrawPublicPoolPublication(ctx, userID, domainID, currentAPIKeyArgs(ctx)...)
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrAPIKeyForbidden):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
		case errors.Is(err, ErrDomainNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrInvalidPublicationState):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to withdraw public pool publication"})
		}
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) ListOwnedProviderAccounts(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	items, err := c.service.ListOwnedProviderAccounts(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list provider accounts"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) CreateOwnedProviderAccount(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	var req CreateProviderAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	item, err := c.service.CreateOwnedProviderAccount(ctx, userID, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create provider account"})
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

func (c *Controller) UpdateOwnedProviderAccount(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	providerAccountID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid provider account id"})
		return
	}

	var req CreateProviderAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	item, err := c.service.UpdateOwnedProviderAccount(ctx, userID, providerAccountID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAccountImmutableFieldsLocked):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update provider account"})
		}
		return
	}

	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) DeleteOwnedProviderAccount(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	providerAccountID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid provider account id"})
		return
	}

	if err := c.service.DeleteOwnedProviderAccount(ctx, userID, providerAccountID); err != nil {
		switch {
		case errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAccountInUse):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete provider account"})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (c *Controller) ValidateOwnedProviderAccount(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	providerAccountID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid provider account id"})
		return
	}

	item, err := c.service.ValidateOwnedProviderAccount(ctx, userID, providerAccountID)
	if err != nil {
		switch {
		case errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAdapterUnavailable):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (c *Controller) ListOwnedProviderZones(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	providerAccountID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid provider account id"})
		return
	}

	items, err := c.service.ListOwnedProviderZones(ctx, userID, providerAccountID)
	if err != nil {
		switch {
		case errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAdapterUnavailable):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) ListOwnedProviderRecords(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	providerAccountID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid provider account id"})
		return
	}

	items, err := c.service.ListOwnedProviderRecords(ctx, userID, providerAccountID, ctx.Param("zoneId"))
	if err != nil {
		switch {
		case errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAdapterUnavailable):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) ListOwnedProviderChangeSets(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	providerAccountID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid provider account id"})
		return
	}

	items, err := c.service.ListOwnedProviderChangeSets(ctx, userID, providerAccountID, ctx.Param("zoneId"))
	if err != nil {
		switch {
		case errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAdapterUnavailable):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list provider change sets"})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) ListOwnedProviderVerifications(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	providerAccountID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid provider account id"})
		return
	}

	items, err := c.service.PreviewOwnedProviderVerifications(ctx, userID, providerAccountID, ctx.Param("zoneId"), ctx.Query("zoneName"))
	if err != nil {
		switch {
		case errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAdapterUnavailable), errors.Is(err, ErrInvalidDNSChangeSetRequest):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list provider verification profiles"})
		}
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items})
}

func (c *Controller) PreviewOwnedProviderChangeSet(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	providerAccountID, ok := domainIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid provider account id"})
		return
	}

	var req PreviewProviderChangeSetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	item, err := c.service.PreviewOwnedProviderChangeSet(ctx, userID, providerAccountID, ctx.Param("zoneId"), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrProviderAccountNotFound), errors.Is(err, ErrDNSChangeSetNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAdapterUnavailable), errors.Is(err, ErrInvalidDNSChangeSetRequest), errors.Is(err, ErrUnsupportedDNSRecordType):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to preview provider change set"})
		}
		return
	}
	ctx.JSON(http.StatusCreated, item)
}

func (c *Controller) ApplyOwnedProviderChangeSet(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	changeSetID, ok := changeSetIDFromParam(ctx)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid change set id"})
		return
	}

	item, err := c.service.ApplyOwnedProviderChangeSet(ctx, userID, changeSetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDNSChangeSetNotFound), errors.Is(err, ErrProviderAccountNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		case errors.Is(err, ErrProviderAdapterUnavailable), errors.Is(err, ErrInvalidDNSChangeSetRequest), errors.Is(err, ErrUnsupportedDNSRecordType):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to apply provider change set"})
		}
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

func domainIDFromParam(ctx *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func changeSetIDFromParam(ctx *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(ctx.Param("changeSetId"), 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func currentAPIKeyArgs(ctx *gin.Context) []portal.APIKey {
	apiKey, ok := middleware.CurrentAPIKey(ctx)
	if !ok {
		return nil
	}
	return []portal.APIKey{apiKey}
}
