package copilot

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	copilotTokenURL        = "https://api.github.com/copilot_internal/v2/token"
	defaultCopilotBaseURL  = "https://api.githubcopilot.com"
	copilotRedisTokenKey   = "copilot:token:"
	copilotTokenEarlySkew  = 5 * time.Minute
	copilotRefreshHeadroom = 60 * time.Second
)

type tokenExchangeResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	RefreshIn int    `json:"refresh_in"`
}

type copilotTokenCache struct {
	Token     string `json:"token"`
	BaseURL   string `json:"base_url"`
	ExpiresAt int64  `json:"expires_at"`
	RefreshIn int    `json:"refresh_in"`
	UpdatedAt int64  `json:"updated_at"`
}

var copilotTokenMemoryCache sync.Map

func GetCopilotToken(githubToken string, accountType string) (string, string, error) {
	githubToken = strings.TrimSpace(resolveGitHubToken(githubToken))
	if githubToken == "" {
		return "", "", fmt.Errorf("copilot channel: missing github token")
	}
	cacheKey := common.Sha1([]byte(githubToken))
	if cached, ok := copilotTokenMemoryCache.Load(cacheKey); ok {
		if token := cached.(*copilotTokenCache); time.Now().UnixMilli() < token.ExpiresAt-copilotTokenEarlySkew.Milliseconds() {
			return token.Token, token.BaseURL, nil
		}
	}
	if common.RedisEnabled {
		if val, err := common.RedisGet(copilotRedisTokenKey + cacheKey); err == nil && val != "" {
			var tokenCache copilotTokenCache
			if err = common.UnmarshalJsonStr(val, &tokenCache); err == nil {
				if time.Now().UnixMilli() < tokenCache.ExpiresAt-copilotTokenEarlySkew.Milliseconds() {
					copilotTokenMemoryCache.Store(cacheKey, &tokenCache)
					return tokenCache.Token, tokenCache.BaseURL, nil
				}
			}
		}
	}
	resp, err := exchangeToken(githubToken)
	if err != nil {
		return "", "", err
	}
	baseURL := resolveCopilotBaseURL(accountType, resp.Token)
	cacheValue := &copilotTokenCache{
		Token:     resp.Token,
		BaseURL:   baseURL,
		ExpiresAt: resp.ExpiresAt * 1000,
		RefreshIn: resp.RefreshIn,
		UpdatedAt: time.Now().UnixMilli(),
	}
	copilotTokenMemoryCache.Store(cacheKey, cacheValue)
	if common.RedisEnabled {
		if raw, marshalErr := common.Marshal(cacheValue); marshalErr == nil {
			ttl := time.Until(time.Unix(resp.ExpiresAt, 0)) - copilotTokenEarlySkew
			if ttl > 0 {
				_ = common.RedisSet(copilotRedisTokenKey+cacheKey, string(raw), ttl)
			}
		}
	}
	if resp.RefreshIn > int(copilotRefreshHeadroom.Seconds()) {
		go scheduleTokenRefresh(cacheKey, githubToken, accountType, resp.RefreshIn)
	}
	return cacheValue.Token, cacheValue.BaseURL, nil
}

func exchangeToken(githubToken string) (*tokenExchangeResponse, error) {
	req, err := http.NewRequest("GET", copilotTokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("editor-version", "vscode/"+GetVSCodeVersion())
	req.Header.Set("editor-plugin-version", defaultCopilotPluginVersion)
	req.Header.Set("user-agent", defaultCopilotUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("copilot token exchange failed: http %d", resp.StatusCode)
	}
	var out tokenExchangeResponse
	if err = common.DecodeJson(resp.Body, &out); err != nil {
		return nil, err
	}
	if out.Token == "" {
		return nil, fmt.Errorf("copilot token exchange failed: empty token")
	}
	return &out, nil
}

func scheduleTokenRefresh(cacheKey, githubToken, accountType string, refreshIn int) {
	time.Sleep(time.Duration(refreshIn)*time.Second - copilotRefreshHeadroom)
	resp, err := exchangeToken(githubToken)
	if err != nil {
		return
	}
	baseURL := resolveCopilotBaseURL(accountType, resp.Token)
	cacheValue := &copilotTokenCache{
		Token:     resp.Token,
		BaseURL:   baseURL,
		ExpiresAt: resp.ExpiresAt * 1000,
		RefreshIn: resp.RefreshIn,
		UpdatedAt: time.Now().UnixMilli(),
	}
	copilotTokenMemoryCache.Store(cacheKey, cacheValue)
	if common.RedisEnabled {
		if raw, marshalErr := common.Marshal(cacheValue); marshalErr == nil {
			ttl := time.Until(time.Unix(resp.ExpiresAt, 0)) - copilotTokenEarlySkew
			if ttl > 0 {
				_ = common.RedisSet(copilotRedisTokenKey+cacheKey, string(raw), ttl)
			}
		}
	}
	if resp.RefreshIn > int(copilotRefreshHeadroom.Seconds()) {
		go scheduleTokenRefresh(cacheKey, githubToken, accountType, resp.RefreshIn)
	}
}

func resolveCopilotBaseURL(accountType, token string) string {
	if parsed := deriveCopilotBaseURLFromToken(token); parsed != "" {
		return parsed
	}
	switch strings.ToLower(strings.TrimSpace(accountType)) {
	case "business":
		return "https://api.business.githubcopilot.com"
	case "enterprise":
		return "https://api.enterprise.githubcopilot.com"
	default:
		return defaultCopilotBaseURL
	}
}

func deriveCopilotBaseURLFromToken(token string) string {
	for _, part := range strings.Split(token, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 || strings.ToLower(kv[0]) != "proxy-ep" {
			continue
		}
		host := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(kv[1])), "https://"), "http://")
		if strings.HasPrefix(host, "proxy.") {
			host = "api." + host[6:]
		}
		if host != "" {
			return "https://" + host
		}
	}
	return ""
}
