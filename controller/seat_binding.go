package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/seat"
	"github.com/gin-gonic/gin"
)

type bindSeatRequest struct {
	TenantID    string `json:"tenant_id"`
	UserID      any    `json:"user_id"`
	SeatID      string `json:"seat_id"`
	GitHubToken string `json:"github_token"`
	AccountType string `json:"account_type"`
}

type seatOAuthStartRequest struct {
	TenantID    string `json:"tenant_id"`
	UserID      any    `json:"user_id"`
	SeatID      string `json:"seat_id"`
	AccountType string `json:"account_type"`
}

type seatOAuthPollRequest struct {
	DeviceCode string `json:"device_code"`
}

type pendingSeatOAuthBind struct {
	TenantID    string
	UserID      int
	SeatID      string
	AccountType string
	ExpiresAt   time.Time
}

var (
	pendingSeatOAuthBinds sync.Map
)

func parseUserID(raw any) (int, error) {
	switch v := raw.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float64:
		if v != float64(int(v)) {
			return 0, fmt.Errorf("user_id must be integer")
		}
		return int(v), nil
	case string:
		if v == "" {
			return 0, fmt.Errorf("user_id is required")
		}
		id, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("user_id must be integer")
		}
		return id, nil
	default:
		return 0, fmt.Errorf("user_id must be integer")
	}
}

func cleanupExpiredSeatOAuthBinds() {
	now := time.Now()
	pendingSeatOAuthBinds.Range(func(k, v any) bool {
		item, ok := v.(pendingSeatOAuthBind)
		if !ok || now.After(item.ExpiresAt) {
			pendingSeatOAuthBinds.Delete(k)
		}
		return true
	})
}

func ListSeatBindings(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListSeatBindings(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, item := range items {
		item.EncryptedToken = "***"
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func CreateOrUpdateSeatBinding(c *gin.Context) {
	var req bindSeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userID, err := parseUserID(req.UserID)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if userID <= 0 || req.SeatID == "" || req.GitHubToken == "" {
		common.ApiErrorMsg(c, "user_id, seat_id, github_token are required")
		return
	}
	if req.AccountType == "" {
		req.AccountType = "individual"
	}
	if err := seat.BindSeat(req.TenantID, userID, req.SeatID, req.GitHubToken, req.AccountType); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"ok": true})
}

func StartSeatBindingGitHubDeviceOAuth(c *gin.Context) {
	var req seatOAuthStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userID, err := parseUserID(req.UserID)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if userID <= 0 || req.SeatID == "" {
		common.ApiErrorMsg(c, "user_id and seat_id are required")
		return
	}
	accountType := req.AccountType
	if accountType == "" {
		accountType = "github_oauth"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	flow, err := service.StartGitHubDeviceFlow(ctx)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanupExpiredSeatOAuthBinds()
	pendingSeatOAuthBinds.Store(flow.DeviceCode, pendingSeatOAuthBind{
		TenantID:    req.TenantID,
		UserID:      userID,
		SeatID:      req.SeatID,
		AccountType: accountType,
		ExpiresAt:   time.Now().Add(time.Duration(flow.ExpiresIn) * time.Second),
	})
	common.ApiSuccess(c, gin.H{
		"device_code":      flow.DeviceCode,
		"user_code":        flow.UserCode,
		"verification_uri": flow.VerificationURI,
		"expires_in":       flow.ExpiresIn,
		"interval":         flow.Interval,
	})
}

func PollSeatBindingGitHubDeviceOAuth(c *gin.Context) {
	var req seatOAuthPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	deviceCode := req.DeviceCode
	if deviceCode == "" {
		common.ApiErrorMsg(c, "device_code is required")
		return
	}
	raw, ok := pendingSeatOAuthBinds.Load(deviceCode)
	if !ok {
		common.ApiErrorMsg(c, "device_code not found or expired")
		return
	}
	bind, ok := raw.(pendingSeatOAuthBind)
	if !ok || time.Now().After(bind.ExpiresAt) {
		pendingSeatOAuthBinds.Delete(deviceCode)
		common.ApiErrorMsg(c, "device_code not found or expired")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	token, err := service.PollGitHubDeviceAccessToken(ctx, deviceCode)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGitHubOAuthPending):
			common.ApiSuccess(c, gin.H{"status": "pending"})
			return
		case errors.Is(err, service.ErrGitHubOAuthSlowDown):
			common.ApiSuccess(c, gin.H{"status": "slow_down"})
			return
		default:
			common.ApiError(c, err)
			return
		}
	}
	if err = seat.BindSeat(bind.TenantID, bind.UserID, bind.SeatID, token.AccessToken, bind.AccountType); err != nil {
		common.ApiError(c, err)
		return
	}
	pendingSeatOAuthBinds.Delete(deviceCode)
	common.ApiSuccess(c, gin.H{
		"ok":         true,
		"status":     "bound",
		"scope":      token.Scope,
		"token_type": token.TokenType,
	})
}

func DeleteSeatBinding(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err = model.DeleteSeatBinding(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"ok": true})
}
