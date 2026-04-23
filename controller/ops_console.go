package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func AdminOperationsOverview(c *gin.Context) {
	var tenantCount int64
	var seatCount int64
	var activeSeatCount int64
	var copilotChannelCount int64
	_ = model.DB.Model(&model.Tenant{}).Count(&tenantCount).Error
	_ = model.DB.Model(&model.SeatBinding{}).Count(&seatCount).Error
	_ = model.DB.Model(&model.SeatBinding{}).Where("status = ?", "active").Count(&activeSeatCount).Error
	_ = model.DB.Model(&model.Channel{}).Where("type = ?", 58).Count(&copilotChannelCount).Error

	var billingRows []struct {
		TenantID     string `json:"tenant_id"`
		InputTokens  int64  `json:"input_tokens"`
		OutputTokens int64  `json:"output_tokens"`
	}
	_ = model.DB.Model(&model.BillingEvent{}).
		Select("tenant_id, sum(input_tokens) as input_tokens, sum(output_tokens) as output_tokens").
		Group("tenant_id").
		Order("sum(input_tokens + output_tokens) desc").
		Limit(10).
		Scan(&billingRows).Error

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tenant_count":          tenantCount,
			"seat_count":            seatCount,
			"active_seat_count":     activeSeatCount,
			"copilot_channel_count": copilotChannelCount,
			"top_tenant_usage":      billingRows,
		},
	})
}
