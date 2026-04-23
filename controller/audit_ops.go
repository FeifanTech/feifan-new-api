package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func parseIntQuery(raw string, defaultValue int) int {
	if raw == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return v
}

func AdminListAuditEvents(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	limit := parseIntQuery(c.Query("limit"), 100)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	query := model.DB.Model(&model.AuditEvent{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	var events []model.AuditEvent
	if err := query.Order("id desc").Limit(limit).Find(&events).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, events)
}

func AdminCleanupAuditEvents(c *gin.Context) {
	days := parseIntQuery(c.Query("days"), 30)
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour).Unix()
	res := model.DB.Where("created_at < ?", cutoff).Delete(&model.AuditEvent{})
	if res.Error != nil {
		common.ApiError(c, res.Error)
		return
	}
	common.ApiSuccess(c, gin.H{"deleted": res.RowsAffected, "cutoff": cutoff})
}

func AdminCopilotTokenHealth(c *gin.Context) {
	var channels []model.Channel
	if err := model.DB.Where("type = ? AND status = ?", constant.ChannelTypeGitHubCopilot, common.ChannelStatusEnabled).Find(&channels).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	type item struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	result := make([]item, 0, len(channels))
	for _, ch := range channels {
		status := "ok"
		key := ch.Key
		if key == "" {
			status = "missing_token"
		}
		result = append(result, item{ID: ch.Id, Name: ch.Name, Status: status})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
