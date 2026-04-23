package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type SeatBinding struct {
	Id             int       `json:"id"`
	TenantID       string    `json:"tenant_id" gorm:"size:64;index:idx_tenant_user,unique"`
	UserID         int       `json:"user_id" gorm:"index:idx_tenant_user,unique"`
	SeatID         string    `json:"seat_id" gorm:"size:128;uniqueIndex"`
	EncryptedToken string    `json:"encrypted_token" gorm:"type:text;not null"`
	AccountType    string    `json:"account_type" gorm:"size:32;default:'individual'"`
	Status         string    `json:"status" gorm:"size:32;default:'active'"`
	BoundAt        time.Time `json:"bound_at"`
	LastUsedAt     time.Time `json:"last_used_at"`
}

func (SeatBinding) TableName() string {
	return "seat_bindings"
}

func GetSeatBindingByTenantAndUser(tenantID string, userID int) (*SeatBinding, error) {
	var binding SeatBinding
	err := DB.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func GetSeatBindingByID(id int) (*SeatBinding, error) {
	var binding SeatBinding
	err := DB.First(&binding, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func ListSeatBindings(offset, limit int) ([]*SeatBinding, int64, error) {
	var items []*SeatBinding
	var total int64
	if err := DB.Model(&SeatBinding{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := DB.Order("id desc").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func CreateSeatBinding(binding *SeatBinding) error {
	if binding == nil {
		return errors.New("binding is nil")
	}
	if binding.BoundAt.IsZero() {
		binding.BoundAt = time.Now()
	}
	return DB.Create(binding).Error
}

func DeleteSeatBinding(id int) error {
	return DB.Delete(&SeatBinding{}, "id = ?", id).Error
}

func UpdateSeatBindingLastUsed(id int) error {
	return DB.Model(&SeatBinding{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error
}

func UpsertSeatBinding(binding *SeatBinding) error {
	if binding == nil {
		return errors.New("binding is nil")
	}
	var existing SeatBinding
	err := DB.Where("tenant_id = ? AND user_id = ?", binding.TenantID, binding.UserID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CreateSeatBinding(binding)
	}
	if err != nil {
		return err
	}
	return DB.Model(&existing).Updates(map[string]any{
		"seat_id":         binding.SeatID,
		"encrypted_token": binding.EncryptedToken,
		"account_type":    binding.AccountType,
		"status":          binding.Status,
		"bound_at":        time.Now(),
	}).Error
}
