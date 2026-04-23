package copilot

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	openaiAdaptor *openai.Adaptor
}

func (a *Adaptor) ensureOpenAIAdaptor(info *relaycommon.RelayInfo) {
	if a.openaiAdaptor == nil {
		a.openaiAdaptor = &openai.Adaptor{}
	}
	a.openaiAdaptor.Init(info)
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	a.ensureOpenAIAdaptor(info)
}

func resolveAccountType(baseURL string) string {
	lower := strings.ToLower(strings.TrimSpace(baseURL))
	if strings.Contains(lower, "business.githubcopilot.com") || lower == "business" {
		return "business"
	}
	if strings.Contains(lower, "enterprise.githubcopilot.com") || lower == "enterprise" {
		return "enterprise"
	}
	return "individual"
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	githubToken := strings.TrimSpace(info.ApiKey)
	copilotToken, baseURL, err := GetCopilotToken(githubToken, resolveAccountType(info.ChannelBaseUrl))
	if err != nil {
		return "", err
	}
	info.ApiKey = copilotToken
	info.ChannelBaseUrl = baseURL

	switch info.RelayMode {
	case relayconstant.RelayModeEmbeddings:
		return baseURL + "/embeddings", nil
	default:
		if strings.HasPrefix(info.RequestURLPath, "/v1/models") {
			return baseURL + "/models", nil
		}
		return baseURL + "/chat/completions", nil
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	if strings.TrimSpace(info.ApiKey) == "" {
		return errors.New("copilot channel: missing copilot token")
	}
	isAgentCall := false
	hasImages := false
	openAIReq := &dto.GeneralOpenAIRequest{}
	if err := common.UnmarshalBodyReusable(c, openAIReq); err == nil {
		for _, msg := range openAIReq.Messages {
			if msg.Role == "assistant" || msg.Role == "tool" {
				isAgentCall = true
			}
			for _, content := range msg.ParseContent() {
				if content.Type == dto.ContentTypeImageURL {
					hasImages = true
				}
			}
		}
	}
	InjectCopilotHeaders(req, info.ApiKey, GetVSCodeVersion(), isAgentCall, hasImages)
	if info.IsStream {
		req.Set("Accept", "text/event-stream")
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	a.ensureOpenAIAdaptor(info)
	request.Model = NormalizeCopilotModelName(request.Model)
	if models, err := FetchCopilotModels(info.ChannelBaseUrl, info.ApiKey); err == nil {
		FillMaxTokens(request, models)
	}
	return a.openaiAdaptor.ConvertOpenAIRequest(c, info, request)
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	a.ensureOpenAIAdaptor(info)
	return a.openaiAdaptor.ConvertClaudeRequest(c, info, request)
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	a.ensureOpenAIAdaptor(info)
	return a.openaiAdaptor.ConvertGeminiRequest(c, info, request)
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("copilot channel: /v1/rerank endpoint not supported")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("copilot channel: endpoint not supported")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("copilot channel: endpoint not supported")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("copilot channel: /v1/responses endpoint not supported")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	a.ensureOpenAIAdaptor(info)
	return a.openaiAdaptor.DoResponse(c, resp, info)
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
