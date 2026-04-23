package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestApiRouterContainsOpsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetApiRouter(r)
	routes := r.Routes()
	routeSet := map[string]bool{}
	for _, rt := range routes {
		routeSet[rt.Method+" "+rt.Path] = true
	}
	require.True(t, routeSet["POST /api/ops/seat/bind"])
	require.True(t, routeSet["GET /api/ops/seat/bindings"])
	require.True(t, routeSet["GET /api/ops/billing/statements"])
	require.True(t, routeSet["GET /api/ops/billing/statements/export"])
	require.True(t, routeSet["GET /api/ops/audit/events"])
	require.True(t, routeSet["DELETE /api/ops/audit/events"])
	require.True(t, routeSet["GET /api/ops/copilot/token-health"])
	require.True(t, routeSet["GET /api/ops/overview"])
}
