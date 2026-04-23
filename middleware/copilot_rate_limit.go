package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

var (
	copilotFallbackMu       sync.Mutex
	copilotSeatConcurrent   = map[string]int{}
	copilotGlobalConcurrent int
	copilotSeatQPMWindow    = map[string]map[int64]int{}
)

func fallbackLimit(v int) int {
	if v <= 1 {
		return 1
	}
	half := v / 2
	if half < 1 {
		return 1
	}
	return half
}

func getSeatIdentity(c *gin.Context) string {
	seatID := c.GetString("seat_id")
	if seatID != "" {
		return seatID
	}
	return fmt.Sprintf("user:%d", c.GetInt("id"))
}

func releaseRedisCounters(seatKey string) {
	if !common.RedisEnabled {
		return
	}
	ctx := context.Background()
	_ = common.RDB.Decr(ctx, seatKey).Err()
	_ = common.RDB.Decr(ctx, "rl:copilot:global:conc").Err()
}

func acquireFallback(seat string, seatConcLimit int, seatQPMLimit int, globalConcLimit int) (bool, int) {
	copilotFallbackMu.Lock()
	defer copilotFallbackMu.Unlock()

	nowMinute := time.Now().Unix() / 60
	if _, ok := copilotSeatQPMWindow[seat]; !ok {
		copilotSeatQPMWindow[seat] = map[int64]int{}
	}
	// cleanup old qpm buckets
	for minute := range copilotSeatQPMWindow[seat] {
		if minute < nowMinute {
			delete(copilotSeatQPMWindow[seat], minute)
		}
	}

	seatConcLimit = fallbackLimit(seatConcLimit)
	seatQPMLimit = fallbackLimit(seatQPMLimit)
	globalConcLimit = fallbackLimit(globalConcLimit)

	if copilotSeatConcurrent[seat] >= seatConcLimit {
		return false, http.StatusTooManyRequests
	}
	if copilotGlobalConcurrent >= globalConcLimit {
		return false, http.StatusServiceUnavailable
	}
	qpm := copilotSeatQPMWindow[seat][nowMinute]
	if qpm >= seatQPMLimit {
		return false, http.StatusTooManyRequests
	}
	copilotSeatConcurrent[seat]++
	copilotGlobalConcurrent++
	copilotSeatQPMWindow[seat][nowMinute] = qpm + 1
	return true, http.StatusOK
}

func releaseFallback(seat string) {
	copilotFallbackMu.Lock()
	defer copilotFallbackMu.Unlock()
	if copilotSeatConcurrent[seat] > 0 {
		copilotSeatConcurrent[seat]--
	}
	if copilotGlobalConcurrent > 0 {
		copilotGlobalConcurrent--
	}
}

func CopilotSeatRateLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		if c.GetInt("channel_type") != constant.ChannelTypeCopilot {
			c.Next()
			return
		}

		seat := getSeatIdentity(c)
		seatConcLimit := common.GetEnvOrDefault("RATE_LIMIT_SEAT_CONCURRENT", 3)
		seatQPMLimit := common.GetEnvOrDefault("RATE_LIMIT_SEAT_QPM", 60)
		globalConcLimit := common.GetEnvOrDefault("RATE_LIMIT_GLOBAL_CONCURRENT", 100)

		if common.RedisEnabled {
			ctx := context.Background()
			seatConcKey := "rl:copilot:seat:" + seat + ":conc"
			seatQPMKey := fmt.Sprintf("rl:copilot:seat:%s:qpm:%d", seat, time.Now().Unix()/60)
			globalConcKey := "rl:copilot:global:conc"

			seatConc, err := common.RDB.Incr(ctx, seatConcKey).Result()
			if err != nil {
				c.Set("copilot_rate_limit_fallback", true)
				ok, status := acquireFallback(seat, seatConcLimit, seatQPMLimit, globalConcLimit)
				if !ok {
					c.AbortWithStatusJSON(status, gin.H{"error": "copilot rate limit exceeded (fallback)"})
					return
				}
				defer releaseFallback(seat)
				c.Next()
				return
			}
			_ = common.RDB.Expire(ctx, seatConcKey, 2*time.Minute).Err()
			if seatConc > int64(seatConcLimit) {
				_ = common.RDB.Decr(ctx, seatConcKey).Err()
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "seat concurrent limit exceeded"})
				return
			}

			globalConc, err := common.RDB.Incr(ctx, globalConcKey).Result()
			if err != nil {
				_ = common.RDB.Decr(ctx, seatConcKey).Err()
				c.Set("copilot_rate_limit_fallback", true)
				ok, status := acquireFallback(seat, seatConcLimit, seatQPMLimit, globalConcLimit)
				if !ok {
					c.AbortWithStatusJSON(status, gin.H{"error": "copilot rate limit exceeded (fallback)"})
					return
				}
				defer releaseFallback(seat)
				c.Next()
				return
			}
			_ = common.RDB.Expire(ctx, globalConcKey, 2*time.Minute).Err()
			if globalConc > int64(globalConcLimit) {
				_ = common.RDB.Decr(ctx, seatConcKey).Err()
				_ = common.RDB.Decr(ctx, globalConcKey).Err()
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "global concurrent limit exceeded"})
				return
			}

			qpm, err := common.RDB.Incr(ctx, seatQPMKey).Result()
			if err != nil {
				releaseRedisCounters(seatConcKey)
				c.Set("copilot_rate_limit_fallback", true)
				ok, status := acquireFallback(seat, seatConcLimit, seatQPMLimit, globalConcLimit)
				if !ok {
					c.AbortWithStatusJSON(status, gin.H{"error": "copilot rate limit exceeded (fallback)"})
					return
				}
				defer releaseFallback(seat)
				c.Next()
				return
			}
			_ = common.RDB.Expire(ctx, seatQPMKey, 70*time.Second).Err()
			if qpm > int64(seatQPMLimit) {
				releaseRedisCounters(seatConcKey)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "seat qpm limit exceeded"})
				return
			}

			defer releaseRedisCounters(seatConcKey)
			c.Next()
			return
		}

		ok, status := acquireFallback(seat, seatConcLimit, seatQPMLimit, globalConcLimit)
		c.Set("copilot_rate_limit_fallback", true)
		if !ok {
			c.AbortWithStatusJSON(status, gin.H{"error": "copilot rate limit exceeded"})
			return
		}
		defer releaseFallback(seat)
		c.Next()
	}
}
