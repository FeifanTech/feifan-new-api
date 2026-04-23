package audit

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var auditEventCh = make(chan *model.AuditEvent, 1024)

func InitAuditProducer(ctx context.Context) {
	go func() {
		batch := make([]*model.AuditEvent, 0, 100)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		flush := func() {
			if len(batch) == 0 {
				return
			}
			values := make([]model.AuditEvent, 0, len(batch))
			for _, event := range batch {
				if event != nil {
					values = append(values, *event)
				}
			}
			if len(values) > 0 {
				_ = model.DB.Create(&values).Error
			}
			batch = batch[:0]
		}
		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case <-ticker.C:
				flush()
			case event := <-auditEventCh:
				if event == nil {
					continue
				}
				batch = append(batch, event)
				if len(batch) >= 100 {
					flush()
				}
			}
		}
	}()
}

func Publish(event *model.AuditEvent) {
	if event == nil {
		return
	}
	select {
	case auditEventCh <- event:
	default:
		// sampling drop when queue is full
		common.SysLog("audit event queue full, dropping event")
	}
}
