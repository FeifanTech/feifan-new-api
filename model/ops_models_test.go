package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupOpsModelDB(t *testing.T) {
	t.Helper()
	oldDB := DB
	t.Cleanup(func() {
		DB = oldDB
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	require.NoError(t, db.AutoMigrate(&Tenant{}, &SeatBinding{}, &BillingEvent{}, &AuditEvent{}))
}

func TestTenantCreateAndGet(t *testing.T) {
	setupOpsModelDB(t)
	tenant := &Tenant{TenantID: "t1", Name: "Tenant One"}
	require.NoError(t, DB.Create(tenant).Error)
	got, err := GetTenantByTenantID("t1")
	require.NoError(t, err)
	require.Equal(t, "Tenant One", got.Name)
	require.Equal(t, "active", got.Status)
	require.NotZero(t, got.CreatedTime)
	require.NotZero(t, got.UpdatedTime)
	require.NoError(t, DB.Model(got).Update("name", "Tenant One Updated").Error)
	refreshed, err := GetTenantByTenantID("t1")
	require.NoError(t, err)
	require.Equal(t, "Tenant One Updated", refreshed.Name)
	require.GreaterOrEqual(t, refreshed.UpdatedTime, got.UpdatedTime)

	_, err = GetTenantByTenantID("missing")
	require.Error(t, err)
}

func TestSeatBindingLookupAndTouch(t *testing.T) {
	setupOpsModelDB(t)
	sb := &SeatBinding{
		TenantID:       "t1",
		UserID:         "u1",
		SeatID:         "s1",
		EncryptedToken: "enc",
	}
	require.NoError(t, DB.Create(sb).Error)
	got, err := GetSeatBindingByTenantAndUser("t1", "u1")
	require.NoError(t, err)
	require.Equal(t, "s1", got.SeatID)
	before := got.LastUsedAt
	got.TouchLastUsed()
	require.GreaterOrEqual(t, got.LastUsedAt, before)
	require.Equal(t, "active", got.Status)
	require.Equal(t, "individual", got.AccountType)
	require.NotZero(t, got.BoundAt)

	_, err = GetSeatBindingByTenantAndUser("t1", "unknown")
	require.Error(t, err)
}

func TestBillingAuditBeforeCreate(t *testing.T) {
	setupOpsModelDB(t)
	b := &BillingEvent{TenantID: "t1", UserID: "u1", SeatID: "s1"}
	require.NoError(t, DB.Create(b).Error)
	require.NotZero(t, b.BilledAt)

	a := &AuditEvent{TenantID: "t1"}
	require.NoError(t, DB.Create(a).Error)
	require.NotZero(t, a.CreatedAt)
	require.NotEmpty(t, common.Sha1([]byte("127.0.0.1")))
}
