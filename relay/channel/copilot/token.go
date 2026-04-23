package copilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/go-redis/redis/v8"
)

const (
	defaultExchangeURL  = "https://api.github.com/copilot_internal/v2/token"
	copilotRedisPrefix  = "copilot:token:"
	redisSafetyLeeway   = 300 * time.Second
	refreshBeforeExpiry = 60 * time.Second
)

type CopilotTokenInfo struct {
	AccessToken string    `json:"access_token"`
	BaseURL     string    `json:"base_url"`
	RefreshIn   int       `json:"refresh_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type tokenCacheItem struct {
	info CopilotTokenInfo
}

var (
	tokenCache      sync.Map
	refreshTaskLock sync.Mutex
	refreshTasks    = make(map[string]struct{})
)

func cacheKey(githubToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(githubToken)))
	return hex.EncodeToString(sum[:])
}

func redisKey(githubToken string) string {
	return copilotRedisPrefix + cacheKey(githubToken)
}

func isValidToken(info CopilotTokenInfo) bool {
	if info.AccessToken == "" {
		return false
	}
	return time.Until(info.ExpiresAt) > redisSafetyLeeway
}

func tokenExchangeURL() string {
	if v := strings.TrimSpace(os.Getenv("COPILOT_TOKEN_EXCHANGE_URL")); v != "" {
		return v
	}
	return defaultExchangeURL
}

func GetEditorVersion() string {
	if v := strings.TrimSpace(os.Getenv("COPILOT_EDITOR_VERSION")); v != "" {
		return v
	}
	return "vscode/1.104.3"
}

func readFromMemory(githubToken string) (CopilotTokenInfo, bool) {
	key := cacheKey(githubToken)
	raw, ok := tokenCache.Load(key)
	if !ok {
		return CopilotTokenInfo{}, false
	}
	item, ok := raw.(tokenCacheItem)
	if !ok || !isValidToken(item.info) {
		tokenCache.Delete(key)
		return CopilotTokenInfo{}, false
	}
	return item.info, true
}

func readFromRedis(githubToken string) (CopilotTokenInfo, bool) {
	if !common.RedisEnabled {
		return CopilotTokenInfo{}, false
	}
	raw, err := common.RedisGet(redisKey(githubToken))
	if err != nil {
		return CopilotTokenInfo{}, false
	}
	info := CopilotTokenInfo{}
	if err = common.UnmarshalJsonStr(raw, &info); err != nil {
		return CopilotTokenInfo{}, false
	}
	if !isValidToken(info) {
		return CopilotTokenInfo{}, false
	}
	return info, true
}

func writeCache(githubToken string, info CopilotTokenInfo) {
	key := cacheKey(githubToken)
	tokenCache.Store(key, tokenCacheItem{info: info})
	if !common.RedisEnabled {
		return
	}
	payload, err := common.Marshal(info)
	if err != nil {
		return
	}
	ttl := time.Until(info.ExpiresAt) - redisSafetyLeeway
	if ttl <= 0 {
		ttl = time.Minute
	}
	_ = common.RedisSet(redisKey(githubToken), string(payload), ttl)
}

func parseBaseURL(payload map[string]any) string {
	if v := common.Interface2String(payload["proxy_endpoint"]); v != "" {
		return strings.Replace(v, "proxy.", "api.", 1)
	}
	if v := common.Interface2String(payload["proxy_ep"]); v != "" {
		return strings.Replace(v, "proxy.", "api.", 1)
	}
	if endpoints, ok := payload["endpoints"].(map[string]any); ok {
		if v := common.Interface2String(endpoints["api"]); v != "" {
			return v
		}
	}
	return ""
}

func exchangeToken(ctx context.Context, githubToken string) (CopilotTokenInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenExchangeURL(), nil)
	if err != nil {
		return CopilotTokenInfo{}, err
	}
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Editor-Version", GetEditorVersion())

	client := service.GetHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return CopilotTokenInfo{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return CopilotTokenInfo{}, fmt.Errorf("copilot token exchange auth failed, status=%d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return CopilotTokenInfo{}, fmt.Errorf("copilot token exchange rate limited, status=429")
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return CopilotTokenInfo{}, fmt.Errorf("copilot token exchange upstream error, status=%d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		if strings.EqualFold(os.Getenv("COPILOT_DEBUG"), "true") {
			common.SysLog(fmt.Sprintf("copilot exchange non-200 status=%d body_len=%d", resp.StatusCode, len(body)))
		}
		return CopilotTokenInfo{}, fmt.Errorf("copilot token exchange failed, status=%d", resp.StatusCode)
	}

	data := map[string]any{}
	if err = common.Unmarshal(body, &data); err != nil {
		return CopilotTokenInfo{}, err
	}

	accessToken := common.Interface2String(data["token"])
	if accessToken == "" {
		accessToken = common.Interface2String(data["access_token"])
	}
	if accessToken == "" {
		return CopilotTokenInfo{}, fmt.Errorf("copilot token exchange missing token")
	}
	refreshIn := 3600
	if v := common.Interface2String(data["refresh_in"]); v != "" {
		if n := common.String2Int(v); n > 0 {
			refreshIn = n
		}
	}
	baseURL := parseBaseURL(data)
	if baseURL == "" {
		baseURL = "https://api.githubcopilot.com"
	}
	if strings.EqualFold(os.Getenv("COPILOT_DEBUG"), "true") {
		common.SysLog(fmt.Sprintf("copilot exchange ok refresh_in=%d base_url=%s", refreshIn, baseURL))
	}
	return CopilotTokenInfo{
		AccessToken: accessToken,
		BaseURL:     baseURL,
		RefreshIn:   refreshIn,
		ExpiresAt:   time.Now().Add(time.Duration(refreshIn) * time.Second),
	}, nil
}

func startRefreshLoop(githubToken string) {
	key := cacheKey(githubToken)
	refreshTaskLock.Lock()
	if _, exists := refreshTasks[key]; exists {
		refreshTaskLock.Unlock()
		return
	}
	refreshTasks[key] = struct{}{}
	refreshTaskLock.Unlock()

	go func() {
		defer func() {
			refreshTaskLock.Lock()
			delete(refreshTasks, key)
			refreshTaskLock.Unlock()
		}()
		for {
			info, ok := readFromMemory(githubToken)
			if !ok {
				return
			}
			wait := time.Duration(info.RefreshIn)*time.Second - refreshBeforeExpiry
			if wait < time.Minute {
				wait = time.Minute
			}
			time.Sleep(wait)

			next, err := exchangeToken(context.Background(), githubToken)
			if err != nil {
				// Keep old cache and retry in a short interval.
				time.Sleep(30 * time.Second)
				continue
			}
			writeCache(githubToken, next)
		}
	}()
}

func GetCopilotAccessToken(ctx context.Context, githubToken string) (CopilotTokenInfo, error) {
	if info, ok := readFromMemory(githubToken); ok {
		return info, nil
	}
	if info, ok := readFromRedis(githubToken); ok {
		writeCache(githubToken, info)
		startRefreshLoop(githubToken)
		return info, nil
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		info, err := exchangeToken(ctx, githubToken)
		if err == nil {
			writeCache(githubToken, info)
			startRefreshLoop(githubToken)
			return info, nil
		}
		lastErr = err
		if strings.Contains(err.Error(), "429") {
			time.Sleep(time.Duration(1<<i) * time.Second)
			continue
		}
		if strings.Contains(err.Error(), "status=5") {
			time.Sleep(time.Duration(1<<i) * time.Second)
			continue
		}
		break
	}

	// 5xx fallback to stale redis value if present.
	if common.RedisEnabled && lastErr != nil && strings.Contains(lastErr.Error(), "status=5") {
		raw, err := common.RedisGet(redisKey(githubToken))
		if err == nil {
			info := CopilotTokenInfo{}
			if common.UnmarshalJsonStr(raw, &info) == nil && info.AccessToken != "" {
				return info, nil
			}
		}
	}
	if lastErr == nil {
		lastErr = redis.Nil
	}
	return CopilotTokenInfo{}, lastErr
}
