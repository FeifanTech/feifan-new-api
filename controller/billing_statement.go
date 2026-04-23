package controller

import (
	"encoding/csv"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type BillingStatementRow struct {
	BillingCycle string `json:"billing_cycle"`
	TenantID     string `json:"tenant_id"`
	RequestCount int64  `json:"request_count"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
}

func AdminGetBillingStatements(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	cycle := c.Query("billing_cycle")
	query := model.DB.Model(&model.BillingEvent{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if cycle != "" {
		query = query.Where("billing_cycle = ?", cycle)
	}
	var rows []BillingStatementRow
	err := query.Select("billing_cycle, tenant_id, count(*) as request_count, sum(input_tokens) as input_tokens, sum(output_tokens) as output_tokens").
		Group("billing_cycle, tenant_id").
		Order("billing_cycle desc").
		Scan(&rows).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}

func AdminExportBillingStatementsCSV(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	cycle := c.Query("billing_cycle")
	query := model.DB.Model(&model.BillingEvent{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if cycle != "" {
		query = query.Where("billing_cycle = ?", cycle)
	}
	var events []model.BillingEvent
	if err := query.Order("id desc").Limit(5000).Find(&events).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="billing_statements.csv"`)
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"id", "tenant_id", "user_id", "seat_id", "request_id", "model", "protocol", "status_code", "input_tokens", "output_tokens", "duration_ms", "billing_cycle"})
	for _, event := range events {
		_ = w.Write([]string{
			strconv.Itoa(event.Id),
			event.TenantID,
			event.UserID,
			event.SeatID,
			event.RequestID,
			event.Model,
			event.Protocol,
			strconv.Itoa(event.StatusCode),
			strconv.Itoa(event.InputTokens),
			strconv.Itoa(event.OutputTokens),
			strconv.Itoa(event.DurationMs),
			event.BillingCycle,
		})
	}
	w.Flush()
}
