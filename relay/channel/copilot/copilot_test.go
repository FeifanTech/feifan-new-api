package copilot

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func withMockHTTPClient(t *testing.T, fn roundTripFunc) {
	original := http.DefaultClient.Transport
	http.DefaultClient.Transport = fn
	t.Cleanup(func() { http.DefaultClient.Transport = original })
}

func TestNormalizeCopilotModelName(t *testing.T) {
	require.Equal(t, "claude-sonnet-4", NormalizeCopilotModelName("claude-sonnet-4-5"))
	require.Equal(t, "claude-opus-4", NormalizeCopilotModelName("claude-opus-4-1"))
	require.Equal(t, "gpt-4o", NormalizeCopilotModelName("gpt-4o"))
}

func TestFillMaxTokens(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{Model: "claude-sonnet-4"}
	models := []CopilotModel{
		{ID: "claude-sonnet-4"},
	}
	models[0].Capabilities.Limits.MaxOutputTokens = 16384
	FillMaxTokens(req, models)
	require.NotNil(t, req.MaxTokens)
	require.EqualValues(t, 16384, *req.MaxTokens)
}

func TestFillMaxTokensFallback(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{Model: "unknown-model"}
	FillMaxTokens(req, nil)
	require.NotNil(t, req.MaxTokens)
	require.EqualValues(t, 8192, *req.MaxTokens)
}

func TestResolveBaseURLFromTokenAndAccountType(t *testing.T) {
	tokenWithProxy := "abc;proxy-ep=proxy.business.githubcopilot.com;foo=bar"
	require.Equal(t, "https://api.business.githubcopilot.com", resolveCopilotBaseURL("individual", tokenWithProxy))
	require.Equal(t, "https://api.enterprise.githubcopilot.com", resolveCopilotBaseURL("enterprise", "no-proxy"))
	require.Equal(t, "https://api.business.githubcopilot.com", resolveCopilotBaseURL("business", "no-proxy"))
	require.Equal(t, "https://api.githubcopilot.com", resolveCopilotBaseURL("individual", "no-proxy"))
}

func TestInjectCopilotHeaders(t *testing.T) {
	h := http.Header{}
	InjectCopilotHeaders(&h, "token123", "1.90.0", true, true)
	require.Equal(t, "Bearer token123", h.Get("Authorization"))
	require.Equal(t, "vscode/1.90.0", h.Get("editor-version"))
	require.Equal(t, "agent", h.Get("X-Initiator"))
	require.Equal(t, "true", h.Get("copilot-vision-request"))
	require.NotEmpty(t, h.Get("x-request-id"))
}

func TestResolveGitHubToken(t *testing.T) {
	t.Setenv("COPILOT_GITHUB_TOKEN", "from-copilot")
	t.Setenv("GH_TOKEN", "from-gh")
	t.Setenv("GITHUB_TOKEN", "from-github")
	require.Equal(t, "inline", resolveGitHubToken("inline"))
	require.Equal(t, "from-copilot", resolveGitHubToken(""))

	_ = os.Unsetenv("COPILOT_GITHUB_TOKEN")
	require.Equal(t, "from-gh", resolveGitHubToken(""))
}

func TestGetVSCodeVersionOverride(t *testing.T) {
	t.Setenv("COPILOT_EDITOR_VERSION", "9.9.9")
	require.Equal(t, "9.9.9", GetVSCodeVersion())
}

func TestGetVSCodeVersionFromCache(t *testing.T) {
	t.Setenv("COPILOT_EDITOR_VERSION", "")
	vsCodeVersionCacheMu.Lock()
	cachedVSCodeVersion = "1.2.3"
	vsCodeVersionCachedAt = time.Now()
	vsCodeVersionCacheMu.Unlock()
	require.Equal(t, "1.2.3", GetVSCodeVersion())
}

func TestGetVSCodeVersionFromFetch(t *testing.T) {
	t.Setenv("COPILOT_EDITOR_VERSION", "")
	vsCodeVersionCacheMu.Lock()
	cachedVSCodeVersion = ""
	vsCodeVersionCachedAt = time.Time{}
	vsCodeVersionCacheMu.Unlock()
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("pkgver=1.106.0\npkgrel=1")),
			Header:     make(http.Header),
		}, nil
	})
	require.Equal(t, "1.106.0", GetVSCodeVersion())
}

func TestFetchLatestVSCodeVersionFallback(t *testing.T) {
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("network down")
	})
	require.Equal(t, vsCodeFallbackVersion, fetchLatestVSCodeVersion())
}

func TestFetchLatestVSCodeVersionSuccess(t *testing.T) {
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("pkgver=1.105.1\npkgrel=1")),
			Header:     make(http.Header),
		}, nil
	})
	require.Equal(t, "1.105.1", fetchLatestVSCodeVersion())
}

func TestFetchCopilotModelsCache(t *testing.T) {
	modelsCache = sync.Map{}
	var calls int
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/models") {
			calls++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-4o","capabilities":{"limits":{"max_output_tokens":4096}}}]}`)),
				Header:     make(http.Header),
			}, nil
		}
		return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
	})
	m1, err := FetchCopilotModels("https://api.githubcopilot.com", "token")
	require.NoError(t, err)
	require.Len(t, m1, 1)
	m2, err := FetchCopilotModels("https://api.githubcopilot.com", "token")
	require.NoError(t, err)
	require.Len(t, m2, 1)
	require.Equal(t, 1, calls)
}

func TestGetCopilotTokenFromExchangeAndCache(t *testing.T) {
	copilotTokenMemoryCache = sync.Map{}
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedisEnabled })
	var calls int
	t.Setenv("COPILOT_EDITOR_VERSION", "1.90.0")
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == copilotTokenURL {
			calls++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(fmt.Sprintf(
					`{"token":"token-abc;proxy-ep=proxy.business.githubcopilot.com","expires_at":%d,"refresh_in":30}`,
					time.Now().Add(30*time.Minute).Unix(),
				))),
				Header: make(http.Header),
			}, nil
		}
		return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
	})
	token, baseURL, err := GetCopilotToken("ghp_xxx", "individual")
	require.NoError(t, err)
	require.Equal(t, "token-abc;proxy-ep=proxy.business.githubcopilot.com", token)
	require.Equal(t, "https://api.business.githubcopilot.com", baseURL)
	_, _, err = GetCopilotToken("ghp_xxx", "individual")
	require.NoError(t, err)
	require.Equal(t, 1, calls)

	_, _, err = GetCopilotToken("", "individual")
	require.Error(t, err)
}

func TestGetCopilotTokenRedisFallbackPath(t *testing.T) {
	copilotTokenMemoryCache = sync.Map{}
	oldRedisEnabled := common.RedisEnabled
	oldRDB := common.RDB
	common.RedisEnabled = true
	common.RDB = redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		common.RDB = oldRDB
	})
	t.Setenv("COPILOT_EDITOR_VERSION", "1.90.0")
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == copilotTokenURL {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(fmt.Sprintf(
					`{"token":"token-redis","expires_at":%d,"refresh_in":61}`,
					time.Now().Add(30*time.Minute).Unix(),
				))),
				Header: make(http.Header),
			}, nil
		}
		return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
	})
	token, _, err := GetCopilotToken("ghp_redis", "individual")
	require.NoError(t, err)
	require.Equal(t, "token-redis", token)
}

func TestExchangeTokenFailures(t *testing.T) {
	t.Setenv("COPILOT_EDITOR_VERSION", "1.90.0")
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader("unauthorized")),
			Header:     make(http.Header),
		}, nil
	})
	_, err := exchangeToken("ghp_xxx")
	require.Error(t, err)

	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"token":"","expires_at":1,"refresh_in":1}`)),
			Header:     make(http.Header),
		}, nil
	})
	_, err = exchangeToken("ghp_xxx")
	require.Error(t, err)
}

func TestAdaptorSimpleMethods(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "business",
		},
	}
	adaptor.Init(info)
	require.Equal(t, "business", resolveAccountType("https://api.business.githubcopilot.com"))
	require.Equal(t, "enterprise", resolveAccountType("enterprise"))
	require.Equal(t, "individual", resolveAccountType(""))

	_, err := adaptor.ConvertRerankRequest(nil, 0, dto.RerankRequest{})
	require.Error(t, err)
	emb, err := adaptor.ConvertEmbeddingRequest(nil, info, dto.EmbeddingRequest{})
	require.NoError(t, err)
	require.NotNil(t, emb)
	_, err = adaptor.ConvertAudioRequest(nil, info, dto.AudioRequest{})
	require.Error(t, err)
	_, err = adaptor.ConvertImageRequest(nil, info, dto.ImageRequest{})
	require.Error(t, err)
	_, err = adaptor.ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{})
	require.Error(t, err)
	require.Equal(t, ChannelName, adaptor.GetChannelName())
	require.NotEmpty(t, adaptor.GetModelList())
}

func TestAdaptorGetRequestURL(t *testing.T) {
	copilotTokenMemoryCache = sync.Map{}
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedisEnabled })
	t.Setenv("COPILOT_EDITOR_VERSION", "1.90.0")
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == copilotTokenURL {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(fmt.Sprintf(
					`{"token":"tok","expires_at":%d,"refresh_in":30}`,
					time.Now().Add(30*time.Minute).Unix(),
				))),
				Header: make(http.Header),
			}, nil
		}
		return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
	})
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:         "ghp_xxx",
			ChannelBaseUrl: "individual",
		},
		RequestURLPath: "/v1/models",
	}
	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.githubcopilot.com/models", url)

	info.RequestURLPath = "/v1/chat/completions"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.githubcopilot.com/chat/completions", url)
}

func TestAdaptorConvertOpenAIRequest(t *testing.T) {
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/models") {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"data":[{"id":"claude-sonnet-4","capabilities":{"limits":{"max_output_tokens":4096}}}]}`,
				)),
				Header: make(http.Header),
			}, nil
		}
		return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
	})
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:         "copilot-token",
			ChannelBaseUrl: "https://api.githubcopilot.com",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "claude-sonnet-4-5",
	}
	converted, err := adaptor.ConvertOpenAIRequest(nil, info, req)
	require.NoError(t, err)
	require.NotNil(t, converted)
	require.Equal(t, "claude-sonnet-4", req.Model)
	require.NotNil(t, req.MaxTokens)
}

func TestAdaptorConvertClaudeAndGemini(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: 0,
		},
	}
	_, _ = adaptor.ConvertClaudeRequest(nil, info, &dto.ClaudeRequest{})
	_, _ = adaptor.ConvertGeminiRequest(nil, info, &dto.GeminiChatRequest{})
}

func TestScheduleTokenRefreshImmediate(t *testing.T) {
	copilotTokenMemoryCache = sync.Map{}
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedisEnabled })
	t.Setenv("COPILOT_EDITOR_VERSION", "1.90.0")
	withMockHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == copilotTokenURL {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(fmt.Sprintf(
					`{"token":"refresh-token","expires_at":%d,"refresh_in":30}`,
					time.Now().Add(30*time.Minute).Unix(),
				))),
				Header: make(http.Header),
			}, nil
		}
		return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
	})
	scheduleTokenRefresh("k1", "ghp_xxx", "individual", 60)
	_, ok := copilotTokenMemoryCache.Load("k1")
	require.True(t, ok)
}

func TestAdaptorSetupRequestHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", http.NoBody)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "copilot-token",
		},
	}
	h := http.Header{}
	adaptor := &Adaptor{}
	err := adaptor.SetupRequestHeader(c, &h, info)
	require.NoError(t, err)
	require.Equal(t, "Bearer copilot-token", h.Get("Authorization"))
	require.NotEmpty(t, h.Get("x-request-id"))
}

func TestAdaptorSetupRequestHeaderAgentAndVision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"gpt-4o","messages":[{"role":"assistant","content":[{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]}]}`
	c, _ := gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		IsStream: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "copilot-token",
		},
	}
	h := http.Header{}
	adaptor := &Adaptor{}
	err := adaptor.SetupRequestHeader(c, &h, info)
	require.NoError(t, err)
	require.Equal(t, "text/event-stream", h.Get("Accept"))
	require.Equal(t, "agent", h.Get("X-Initiator"))
	require.Equal(t, "true", h.Get("copilot-vision-request"))
}
