package model

import "github.com/QuantumNous/new-api/common"
import "gorm.io/gorm"

type AuditEvent struct {
	Id           int    `json:"id"`
	TenantID     string `json:"tenant_id" gorm:"type:varchar(64);index:idx_audit_tenant_time,priority:1;not null"`
	UserID       string `json:"user_id" gorm:"type:varchar(128)"`
	SeatID       string `json:"seat_id" gorm:"type:varchar(128)"`
	Model        string `json:"model" gorm:"type:varchar(64)"`
	Protocol     string `json:"protocol" gorm:"type:varchar(16)"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	StatusCode   int    `json:"status_code"`
	DurationMs   int    `json:"duration_ms"`
	ClientIPHash string `json:"client_ip_hash" gorm:"type:varchar(64)"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index:idx_audit_tenant_time,priority:2"`
}

func (a *AuditEvent) BeforeCreate(tx *gorm.DB) error {
	a.CreatedAt = common.GetTimestamp()
	return nil
}
