package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/seat"
	"github.com/gin-gonic/gin"
)

type BindSeatRequest struct {
	TenantID    string `json:"tenant_id" binding:"required"`
	UserID      string `json:"user_id" binding:"required"`
	SeatID      string `json:"seat_id" binding:"required"`
	GitHubToken string `json:"github_token" binding:"required"`
	AccountType string `json:"account_type"`
}

func AdminBindSeat(c *gin.Context) {
	req := &BindSeatRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.AccountType == "" {
		req.AccountType = "individual"
	}
	binding, err := seat.BindSeat(req.TenantID, req.UserID, req.SeatID, req.GitHubToken, req.AccountType)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, binding)
}

func AdminListSeatBindings(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	query := model.DB.Model(&model.SeatBinding{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	var bindings []model.SeatBinding
	if err := query.Order("id desc").Limit(200).Find(&bindings).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	// never return encrypted token
	for i := range bindings {
		bindings[i].EncryptedToken = ""
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bindings})
}
