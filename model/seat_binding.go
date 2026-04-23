package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type SeatBinding struct {
	Id             int    `json:"id"`
	TenantID       string `json:"tenant_id" gorm:"type:varchar(64);index:idx_seat_lookup,priority:1;not null"`
	UserID         string `json:"user_id" gorm:"type:varchar(128);index:idx_seat_lookup,priority:2;not null"`
	SeatID         string `json:"seat_id" gorm:"type:varchar(128);uniqueIndex;not null"`
	EncryptedToken string `json:"encrypted_token" gorm:"type:text;not null"`
	AccountType    string `json:"account_type" gorm:"type:varchar(32);default:'individual'"`
	Status         string `json:"status" gorm:"type:varchar(32);default:'active'"`
	BoundAt        int64  `json:"bound_at" gorm:"bigint"`
	LastUsedAt     int64  `json:"last_used_at" gorm:"bigint"`
}

func (s *SeatBinding) BeforeCreate(tx *gorm.DB) error {
	s.BoundAt = common.GetTimestamp()
	if s.Status == "" {
		s.Status = "active"
	}
	if s.AccountType == "" {
		s.AccountType = "individual"
	}
	return nil
}

func (s *SeatBinding) TouchLastUsed() {
	s.LastUsedAt = common.GetTimestamp()
	_ = DB.Model(s).Update("last_used_at", s.LastUsedAt).Error
}

func GetSeatBindingByTenantAndUser(tenantID, userID string) (*SeatBinding, error) {
	binding := &SeatBinding{}
	if err := DB.Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, "active").First(binding).Error; err != nil {
		return nil, err
	}
	return binding, nil
}
