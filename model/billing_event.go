package model

import "github.com/QuantumNous/new-api/common"
import "gorm.io/gorm"

type BillingEvent struct {
	Id           int    `json:"id"`
	TenantID     string `json:"tenant_id" gorm:"type:varchar(64);index:idx_billing_tenant_cycle,priority:1;not null"`
	UserID       string `json:"user_id" gorm:"type:varchar(128);not null"`
	SeatID       string `json:"seat_id" gorm:"type:varchar(128);not null"`
	RequestID    string `json:"request_id" gorm:"type:varchar(128);uniqueIndex"`
	Model        string `json:"model" gorm:"type:varchar(64)"`
	Protocol     string `json:"protocol" gorm:"type:varchar(16)"`
	StatusCode   int    `json:"status_code"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	DurationMs   int    `json:"duration_ms"`
	BilledAt     int64  `json:"billed_at" gorm:"bigint"`
	BillingCycle string `json:"billing_cycle" gorm:"type:varchar(7);index:idx_billing_tenant_cycle,priority:2"`
}

func (b *BillingEvent) BeforeCreate(tx *gorm.DB) error {
	b.BilledAt = common.GetTimestamp()
	return nil
}
