package audit

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

var (
	startOnce sync.Once
	eventCh   chan model.AuditEvent
	metrics   = &metricsAccumulator{}
)

type metricsAccumulator struct {
	mu           sync.Mutex
	total        int
	count403     int
	count429     int
	fallbackHits int
	durations    []int64
}

type metricsSnapshot struct {
	total        int
	count403     int
	count429     int
	fallbackHits int
	p95ms        int64
}

func start() {
	startOnce.Do(func() {
		eventCh = make(chan model.AuditEvent, 2048)
		go consume()
		go startRetentionLoop()
		go startMetricsLoop()
	})
}

func consume() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	batch := make([]model.AuditEvent, 0, 100)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		_ = model.DB.Create(&batch).Error
		batch = batch[:0]
	}
	for {
		select {
		case e := <-eventCh:
			batch = append(batch, e)
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (m *metricsAccumulator) add(status int, durationMS int64, fallback bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.total++
	if status == 403 {
		m.count403++
	}
	if status == 429 {
		m.count429++
	}
	if fallback {
		m.fallbackHits++
	}
	m.durations = append(m.durations, durationMS)
}

func (m *metricsAccumulator) snapshotAndReset() metricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := metricsSnapshot{
		total:        m.total,
		count403:     m.count403,
		count429:     m.count429,
		fallbackHits: m.fallbackHits,
	}
	if len(m.durations) > 0 {
		durations := append([]int64(nil), m.durations...)
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		idx := int(float64(len(durations)-1) * 0.95)
		if idx < 0 {
			idx = 0
		}
		snapshot.p95ms = durations[idx]
	}

	m.total = 0
	m.count403 = 0
	m.count429 = 0
	m.fallbackHits = 0
	m.durations = m.durations[:0]
	return snapshot
}

func startRetentionLoop() {
	retentionDays := common.GetEnvOrDefault("AUDIT_RETENTION_DAYS", 7)
	intervalMinutes := common.GetEnvOrDefault("AUDIT_CLEANUP_INTERVAL_MINUTES", 60)
	if retentionDays <= 0 || intervalMinutes <= 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
		if err := model.DB.Where("created_at < ?", cutoff).Delete(&model.AuditEvent{}).Error; err != nil {
			common.SysError("audit cleanup failed: " + err.Error())
		}
	}
}

func startMetricsLoop() {
	intervalSeconds := common.GetEnvOrDefault("AUDIT_METRICS_INTERVAL_SECONDS", 60)
	if intervalSeconds <= 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	threshold403Rate := float64(common.GetEnvOrDefault("AUDIT_ALERT_403_RATE_PERCENT", 5)) / 100.0
	threshold429Rate := float64(common.GetEnvOrDefault("AUDIT_ALERT_429_RATE_PERCENT", 10)) / 100.0
	thresholdP95MS := int64(common.GetEnvOrDefault("AUDIT_ALERT_P95_MS", 3000))
	thresholdFallbackHits := common.GetEnvOrDefault("AUDIT_ALERT_FALLBACK_HITS", 1)

	for range ticker.C {
		s := metrics.snapshotAndReset()
		if s.total == 0 {
			continue
		}
		rate403 := float64(s.count403) / float64(s.total)
		rate429 := float64(s.count429) / float64(s.total)
		common.SysLog(fmt.Sprintf("audit_metrics window total=%d 403=%d 429=%d p95_ms=%d fallback_hits=%d", s.total, s.count403, s.count429, s.p95ms, s.fallbackHits))
		if rate403 >= threshold403Rate || rate429 >= threshold429Rate || s.p95ms >= thresholdP95MS || s.fallbackHits >= thresholdFallbackHits {
			common.SysError(fmt.Sprintf("audit_alert triggered: total=%d 403_rate=%.4f 429_rate=%.4f p95_ms=%d fallback_hits=%d", s.total, rate403, rate429, s.p95ms, s.fallbackHits))
		}
	}
}

func detectProtocol(info *relaycommon.RelayInfo) string {
	if info == nil {
		return "openai"
	}
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		return "anthropic"
	default:
		return "openai"
	}
}

func safeDurationMS(c *gin.Context) int64 {
	startAt := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
	if startAt.IsZero() {
		return 0
	}
	return time.Since(startAt).Milliseconds()
}

func RecordRelayEvent(c *gin.Context, info *relaycommon.RelayInfo, relayErr *types.NewAPIError) {
	start()
	status := 200
	if relayErr != nil {
		status = relayErr.StatusCode
	}
	modelName := ""
	inTokens := 0
	outTokens := 0
	tenantID := c.GetHeader("X-Tenant-Id")
	if tenantID == "" {
		tenantID = "default"
	}
	if info != nil {
		modelName = info.OriginModelName
		inTokens = info.GetEstimatePromptTokens()
		outTokens = 0
	}
	seatID := c.GetString("seat_id")
	if seatID == "" && c.GetInt("channel_type") == constant.ChannelTypeCopilot {
		if binding, err := model.GetSeatBindingByTenantAndUser(tenantID, c.GetInt("id")); err == nil {
			seatID = binding.SeatID
		}
	}
	requestID := c.GetString(common.RequestIdKey)
	fallbackUsed := c.GetBool("copilot_rate_limit_fallback")
	event := model.AuditEvent{
		RequestID:    requestID,
		TenantID:     tenantID,
		UserID:       c.GetInt("id"),
		SeatID:       seatID,
		Model:        modelName,
		Protocol:     detectProtocol(info),
		InputTokens:  inTokens,
		OutputTokens: outTokens,
		StatusCode:   status,
		DurationMS:   safeDurationMS(c),
		ClientIPHash: common.GenerateHMAC(c.ClientIP()),
		CreatedAt:    time.Now(),
	}
	select {
	case eventCh <- event:
	default:
		// Never block the relay path on audit.
	}
	metrics.add(status, event.DurationMS, fallbackUsed)
}
