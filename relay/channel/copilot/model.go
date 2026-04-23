package copilot

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

type CopilotModel struct {
	ID           string `json:"id"`
	Capabilities struct {
		Limits struct {
			MaxOutputTokens int `json:"max_output_tokens"`
		} `json:"limits"`
	} `json:"capabilities"`
}

type copilotModelsCache struct {
	Models   []CopilotModel
	CachedAt time.Time
}

var modelsCache sync.Map

func NormalizeCopilotModelName(modelID string) string {
	lower := strings.ToLower(modelID)
	if strings.HasPrefix(lower, "claude-sonnet-4-") {
		return "claude-sonnet-4"
	}
	if strings.HasPrefix(lower, "claude-opus-4-") {
		return "claude-opus-4"
	}
	return modelID
}

func FetchCopilotModels(baseURL, copilotToken string) ([]CopilotModel, error) {
	if cached, ok := modelsCache.Load(baseURL); ok {
		c := cached.(*copilotModelsCache)
		if time.Since(c.CachedAt) < 30*time.Minute {
			return c.Models, nil
		}
	}
	req, err := http.NewRequest("GET", baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+copilotToken)
	req.Header.Set("Accept", "application/json")
	var out struct {
		Data []CopilotModel `json:"data"`
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err = common.DecodeJson(resp.Body, &out); err != nil {
		return nil, err
	}
	modelsCache.Store(baseURL, &copilotModelsCache{Models: out.Data, CachedAt: time.Now()})
	return out.Data, nil
}

func FillMaxTokens(payload *dto.GeneralOpenAIRequest, models []CopilotModel) {
	if payload.MaxTokens != nil && *payload.MaxTokens > 0 {
		return
	}
	for _, model := range models {
		if model.ID != payload.Model {
			continue
		}
		if model.Capabilities.Limits.MaxOutputTokens > 0 {
			maxTokens := uint(model.Capabilities.Limits.MaxOutputTokens)
			payload.MaxTokens = &maxTokens
			return
		}
	}
	defaultMax := uint(8192)
	payload.MaxTokens = &defaultMax
}
