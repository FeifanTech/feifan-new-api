package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

var (
	localSeatConcurrent sync.Map
	localGlobalCounter  int64
	localCounterMutex   sync.Mutex
)

func SeatRateLimit() func(c *gin.Context) {
	seatConcLimit := common.GetEnvOrDefault("RATE_LIMIT_SEAT_CONCURRENT", 3)
	seatQPM := common.GetEnvOrDefault("RATE_LIMIT_SEAT_QPM", 60)
	globalConcLimit := common.GetEnvOrDefault("RATE_LIMIT_GLOBAL_CONCURRENT", 100)
	return func(c *gin.Context) {
		seatID := common.GetContextKeyString(c, constant.ContextKeySeatID)
		if seatID == "" {
			c.Next()
			return
		}
		if common.RedisEnabled {
			if !acquireRedisSeatLimit(c, seatID, seatConcLimit, seatQPM, globalConcLimit) {
				return
			}
			defer releaseRedisSeatLimit(seatID)
		} else {
			if !acquireLocalSeatLimit(c, seatID, seatConcLimit, globalConcLimit) {
				return
			}
			defer releaseLocalSeatLimit(seatID)
		}
		c.Next()
	}
}

func acquireRedisSeatLimit(c *gin.Context, seatID string, seatConcLimit, seatQPM, globalConcLimit int) bool {
	ctx := context.Background()
	seatConcKey := "rl:seat:" + seatID + ":conc"
	seatQPMKey := "rl:seat:" + seatID + ":qpm:" + strconv.FormatInt(time.Now().Unix()/60, 10)
	globalConcKey := "rl:global:conc"

	seatConc, _ := common.RDB.Get(ctx, seatConcKey).Int()
	if seatConc >= seatConcLimit {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "SEAT_CONCURRENT"})
		return false
	}
	seatMinuteCount, _ := common.RDB.Get(ctx, seatQPMKey).Int()
	if seatMinuteCount >= seatQPM {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "SEAT_QPM"})
		return false
	}
	globalConc, _ := common.RDB.Get(ctx, globalConcKey).Int()
	if globalConc >= globalConcLimit {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "GLOBAL_OVERLOAD"})
		return false
	}
	_ = common.RDB.Incr(ctx, seatConcKey).Err()
	_ = common.RDB.Expire(ctx, seatConcKey, 2*time.Minute).Err()
	_ = common.RDB.Incr(ctx, seatQPMKey).Err()
	_ = common.RDB.Expire(ctx, seatQPMKey, 2*time.Minute).Err()
	_ = common.RDB.Incr(ctx, globalConcKey).Err()
	_ = common.RDB.Expire(ctx, globalConcKey, 2*time.Minute).Err()
	return true
}

func releaseRedisSeatLimit(seatID string) {
	ctx := context.Background()
	_ = common.RDB.Decr(ctx, "rl:seat:"+seatID+":conc").Err()
	_ = common.RDB.Decr(ctx, "rl:global:conc").Err()
}

func acquireLocalSeatLimit(c *gin.Context, seatID string, seatConcLimit, globalConcLimit int) bool {
	localCounterMutex.Lock()
	defer localCounterMutex.Unlock()
	raw, _ := localSeatConcurrent.LoadOrStore(seatID, int64(0))
	seatCount := raw.(int64)
	if int(seatCount) >= seatConcLimit {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "SEAT_CONCURRENT"})
		return false
	}
	if int(localGlobalCounter) >= globalConcLimit/2 {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "GLOBAL_OVERLOAD"})
		return false
	}
	localSeatConcurrent.Store(seatID, seatCount+1)
	localGlobalCounter++
	return true
}

func releaseLocalSeatLimit(seatID string) {
	localCounterMutex.Lock()
	defer localCounterMutex.Unlock()
	raw, ok := localSeatConcurrent.Load(seatID)
	if ok {
		count := raw.(int64)
		if count <= 1 {
			localSeatConcurrent.Delete(seatID)
		} else {
			localSeatConcurrent.Store(seatID, count-1)
		}
	}
	if localGlobalCounter > 0 {
		localGlobalCounter--
	}
}
