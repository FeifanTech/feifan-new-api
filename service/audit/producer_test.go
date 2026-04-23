package audit

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPublishAndFlushAuditEvent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.AuditEvent{}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	InitAuditProducer(ctx)

	Publish(&model.AuditEvent{
		TenantID: "t1",
		UserID:   "u1",
		Model:    "gpt-4o",
	})

	time.Sleep(6 * time.Second)
	var count int64
	require.NoError(t, model.DB.Model(&model.AuditEvent{}).Count(&count).Error)
	require.GreaterOrEqual(t, count, int64(1))
}

func TestPublishQueueBranches(t *testing.T) {
	original := auditEventCh
	auditEventCh = make(chan *model.AuditEvent, 1)
	t.Cleanup(func() {
		auditEventCh = original
	})

	// nil event should be ignored
	Publish(nil)

	// fill queue and force default branch (drop path)
	Publish(&model.AuditEvent{TenantID: "t1"})
	Publish(&model.AuditEvent{TenantID: "t2"})
}
