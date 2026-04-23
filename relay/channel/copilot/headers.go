package copilot

import (
	"net/http"
	"os"

	"github.com/QuantumNous/new-api/common"
)

const (
	defaultCopilotPluginVersion = "copilot-chat/0.26.7"
	defaultCopilotUserAgent     = "GitHubCopilotChat/0.26.7"
	defaultGitHubAPIVersion     = "2025-04-01"
	defaultUserAgentLibrary     = "electron-fetch"
)

func InjectCopilotHeaders(req *http.Header, copilotToken, vsCodeVersion string, isAgentCall, hasImages bool) {
	req.Set("Authorization", "Bearer "+copilotToken)
	req.Set("Content-Type", "application/json")
	req.Set("Accept", "application/json")
	req.Set("editor-version", "vscode/"+vsCodeVersion)
	req.Set("editor-plugin-version", common.GetEnvOrDefaultString("COPILOT_PLUGIN_VERSION", defaultCopilotPluginVersion))
	req.Set("user-agent", common.GetEnvOrDefaultString("COPILOT_USER_AGENT", defaultCopilotUserAgent))
	req.Set("copilot-integration-id", "vscode-chat")
	req.Set("openai-intent", "conversation-panel")
	req.Set("x-github-api-version", common.GetEnvOrDefaultString("COPILOT_GITHUB_API_VERSION", defaultGitHubAPIVersion))
	req.Set("x-request-id", common.GetUUID())
	req.Set("x-vscode-user-agent-library-version", defaultUserAgentLibrary)
	if isAgentCall {
		req.Set("X-Initiator", "agent")
	} else {
		req.Set("X-Initiator", "user")
	}
	if hasImages {
		req.Set("copilot-vision-request", "true")
	}
}

func resolveGitHubToken(channelToken string) string {
	if channelToken != "" {
		return channelToken
	}
	for _, envName := range []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if v := os.Getenv(envName); v != "" {
			return v
		}
	}
	return ""
}
