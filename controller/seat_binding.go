package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/seat"
	"github.com/gin-gonic/gin"
)

type bindSeatRequest struct {
	TenantID    string `json:"tenant_id"`
	UserID      int    `json:"user_id"`
	SeatID      string `json:"seat_id"`
	GitHubToken string `json:"github_token"`
	AccountType string `json:"account_type"`
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
	if req.UserID <= 0 || req.SeatID == "" || req.GitHubToken == "" {
		common.ApiErrorMsg(c, "user_id, seat_id, github_token are required")
		return
	}
	if req.AccountType == "" {
		req.AccountType = "individual"
	}
	if err := seat.BindSeat(req.TenantID, req.UserID, req.SeatID, req.GitHubToken, req.AccountType); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"ok": true})
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
