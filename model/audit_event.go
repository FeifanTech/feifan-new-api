package model

import "time"

type AuditEvent struct {
	Id           int       `json:"id"`
	RequestID    string    `json:"request_id" gorm:"size:128;index"`
	TenantID     string    `json:"tenant_id" gorm:"size:64;index"`
	UserID       int       `json:"user_id" gorm:"index"`
	SeatID       string    `json:"seat_id" gorm:"size:128;index"`
	Model        string    `json:"model" gorm:"size:128;index"`
	Protocol     string    `json:"protocol" gorm:"size:32"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	StatusCode   int       `json:"status_code" gorm:"index"`
	DurationMS   int64     `json:"duration_ms"`
	ClientIPHash string    `json:"client_ip_hash" gorm:"size:128"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}

func (AuditEvent) TableName() string {
	return "audit_events"
}
