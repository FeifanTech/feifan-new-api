package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestAcquireAndReleaseLocalSeatLimit(t *testing.T) {
	localSeatConcurrent = syncMapReset()
	localGlobalCounter = 0
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ok := acquireLocalSeatLimit(c, "seat-1", 1, 10)
	require.True(t, ok)
	ok = acquireLocalSeatLimit(c, "seat-1", 1, 10)
	require.False(t, ok)
	releaseLocalSeatLimit("seat-1")
	ok = acquireLocalSeatLimit(c, "seat-1", 1, 10)
	require.True(t, ok)
}

func TestSeatRateLimitMiddlewareNoSeat(t *testing.T) {
	common.RedisEnabled = false
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SeatRateLimit())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSeatRateLimitMiddlewareWithSeat(t *testing.T) {
	common.RedisEnabled = false
	localSeatConcurrent = syncMapReset()
	localGlobalCounter = 0
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeySeatID, "seat-a")
		c.Next()
	})
	r.Use(SeatRateLimit())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAcquireLocalSeatLimitGlobalOverload(t *testing.T) {
	localSeatConcurrent = syncMapReset()
	localGlobalCounter = 5
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ok := acquireLocalSeatLimit(c, "seat-1", 10, 10)
	require.False(t, ok)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestAcquireAndReleaseRedisSeatLimitFallback(t *testing.T) {
	common.RDB = redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ok := acquireRedisSeatLimit(c, "seat-r", 1, 1, 1)
	require.True(t, ok)
	releaseRedisSeatLimit("seat-r")
}

func syncMapReset() (m sync.Map) {
	return
}
