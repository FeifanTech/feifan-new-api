package copilot

import (
	"net/http"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

const ChannelName = "copilot"

type Adaptor struct {
	openai.Adaptor
}

const contextKeyCopilotToken = "copilot_access_token"

func normalizeClaudeModel(model string) string {
	// Copilot endpoint usually rejects Claude sub-version suffixes.
	if strings.HasPrefix(model, "claude-sonnet-4") {
		return "claude-sonnet-4"
	}
	if strings.HasPrefix(model, "claude-opus-4") {
		return "claude-opus-4"
	}
	if strings.HasPrefix(model, "claude-haiku-4") {
		return "claude-haiku-4"
	}
	return model
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	a.Adaptor.Init(info)
	a.ChannelType = constant.ChannelTypeCopilot
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, header *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, header)
	copilotToken := c.GetString(contextKeyCopilotToken)
	if copilotToken == "" {
		tokenInfo, err := GetCopilotAccessToken(c.Request.Context(), info.ApiKey)
		if err != nil {
			return err
		}
		copilotToken = tokenInfo.AccessToken
		c.Set(contextKeyCopilotToken, copilotToken)
	}
	header.Set("Authorization", "Bearer "+copilotToken)
	header.Set("editor-version", GetEditorVersion())
	header.Set("editor-plugin-version", "copilot-chat/0.26.7")
	header.Set("user-agent", "GitHubCopilotChat/0.26.7")
	header.Set("copilot-integration-id", "vscode-chat")
	header.Set("openai-intent", "conversation-panel")
	header.Set("x-github-api-version", "2025-04-01")
	header.Set("x-request-id", common.GetUUID())
	return nil
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	tokenInfo, err := GetCopilotAccessToken(nil, info.ApiKey)
	if err != nil {
		return "", err
	}
	baseURL := info.ChannelBaseUrl
	if tokenInfo.BaseURL != "" {
		baseURL = tokenInfo.BaseURL
		info.ChannelBaseUrl = tokenInfo.BaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := requestURLByMode(baseURL, info.RelayMode)
	if strings.EqualFold(os.Getenv("COPILOT_DEBUG"), "true") {
		common.SysLog("copilot request url: " + url)
	}
	return url, nil
}

func requestURLByMode(baseURL string, relayMode int) string {
	// Copilot upstream paths are chat/completions and embeddings without /v1 prefix.
	switch relayMode {
	case relayconstant.RelayModeEmbeddings:
		return baseURL + "/embeddings"
	default:
		return baseURL + "/chat/completions"
	}
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request != nil {
		request.Model = normalizeClaudeModel(request.Model)
	}
	return a.Adaptor.ConvertOpenAIRequest(c, info, request)
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	if request != nil {
		request.Model = normalizeClaudeModel(request.Model)
	}
	return a.Adaptor.ConvertClaudeRequest(c, info, request)
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
