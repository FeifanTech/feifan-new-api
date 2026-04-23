package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func TestCopilotSeatRateLimitFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = os.Setenv("RATE_LIMIT_SEAT_CONCURRENT", "1")
	_ = os.Setenv("RATE_LIMIT_SEAT_QPM", "1")
	_ = os.Setenv("RATE_LIMIT_GLOBAL_CONCURRENT", "1")
	common.RedisEnabled = false

	handler := CopilotSeatRateLimit()

	run := func() int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		c.Request = req
		c.Set("channel_type", constant.ChannelTypeCopilot)
		c.Set("id", 1001)
		c.Set("seat_id", "seat-a")
		handler(c)
		if c.IsAborted() {
			return w.Code
		}
		return http.StatusOK
	}

	first := run()
	if first != http.StatusOK {
		t.Fatalf("first request got status %d, want 200", first)
	}
	second := run()
	if second != http.StatusTooManyRequests {
		t.Fatalf("second request got status %d, want 429", second)
	}
}
