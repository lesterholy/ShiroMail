package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service *Service
}

func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

func (c *Controller) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.Register(ctx, req)
	if err != nil {
		var pending *PendingVerificationError
		if errors.As(err, &pending) {
			ctx.JSON(http.StatusAccepted, pending.Challenge)
			return
		}
		ctx.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, result)
}

func (c *Controller) Settings(ctx *gin.Context) {
	result, err := c.service.Settings(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load auth settings"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.Login(ctx, req)
	if err != nil {
		var pending *PendingVerificationError
		if errors.As(err, &pending) {
			ctx.JSON(http.StatusForbidden, pending.Challenge)
			return
		}
		var pendingMFA *PendingMFAError
		if errors.As(err, &pendingMFA) {
			ctx.JSON(http.StatusForbidden, pendingMFA.Challenge)
			return
		}
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) Refresh(ctx *gin.Context) {
	var req RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.Refresh(ctx, req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "invalid refresh token"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) Logout(ctx *gin.Context) {
	var req LogoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := c.service.Logout(ctx, req.RefreshToken); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "invalid refresh token"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (c *Controller) ForgotPassword(ctx *gin.Context) {
	var req ForgotPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.ForgotPassword(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) ResetPassword(ctx *gin.Context) {
	var req ResetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.ResetPassword(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) StartOAuth(ctx *gin.Context) {
	provider := ctx.Param("provider")
	result, err := c.service.StartOAuth(ctx, provider)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) CompleteOAuth(ctx *gin.Context) {
	provider := ctx.Param("provider")
	var req OAuthCallbackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.CompleteOAuth(ctx, provider, req)
	if err != nil {
		var pending *PendingVerificationError
		if errors.As(err, &pending) {
			ctx.JSON(http.StatusAccepted, pending.Challenge)
			return
		}
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) ConfirmEmailVerification(ctx *gin.Context) {
	var req EmailVerificationConfirmRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.ConfirmEmailVerification(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) ResendEmailVerification(ctx *gin.Context) {
	var req EmailVerificationResendRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.ResendEmailVerification(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) GetAccountProfile(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	result, err := c.service.GetAccountProfile(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load account profile"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) UpdateAccountProfile(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req UpdateAccountProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.UpdateAccountProfile(ctx, userID, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update account profile"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) RequestEmailChange(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req RequestEmailChangeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.RequestEmailChange(ctx, userID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) ConfirmEmailChange(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req ConfirmEmailChangeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.ConfirmEmailChange(ctx, userID, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) ChangePassword(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := c.service.ChangePassword(ctx, userID, req); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (c *Controller) GetTOTPStatus(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	result, err := c.service.GetTOTPStatus(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load two factor status"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) SetupTOTP(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	result, err := c.service.SetupTOTP(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to setup two factor"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *Controller) EnableTOTP(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req EnableTOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := c.service.EnableTOTP(ctx, userID, req); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (c *Controller) DisableTOTP(ctx *gin.Context) {
	userID, ok := authUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req DisableTOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := c.service.DisableTOTP(ctx, userID, req); err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (c *Controller) VerifyLoginTOTP(ctx *gin.Context) {
	var req VerifyLoginTOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	result, err := c.service.VerifyLoginTOTP(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func authUserID(ctx *gin.Context) (uint64, bool) {
	value, ok := ctx.Get("auth.userID")
	if !ok {
		return 0, false
	}
	userID, ok := value.(uint64)
	return userID, ok
}
