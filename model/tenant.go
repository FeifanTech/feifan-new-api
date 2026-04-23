package model

import "github.com/QuantumNous/new-api/common"
import "gorm.io/gorm"

type Tenant struct {
	Id          int    `json:"id"`
	TenantID    string `json:"tenant_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	Name        string `json:"name" gorm:"type:varchar(128);not null"`
	Status      string `json:"status" gorm:"type:varchar(32);default:'active'"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	t.CreatedTime = now
	t.UpdatedTime = now
	if t.Status == "" {
		t.Status = "active"
	}
	return nil
}

func (t *Tenant) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedTime = common.GetTimestamp()
	return nil
}

func GetTenantByTenantID(tenantID string) (*Tenant, error) {
	tenant := &Tenant{}
	if err := DB.Where("tenant_id = ?", tenantID).First(tenant).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}
