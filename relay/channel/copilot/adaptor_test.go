package copilot

import (
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestNormalizeClaudeModel(t *testing.T) {
	cases := map[string]string{
		"claude-sonnet-4":            "claude-sonnet-4",
		"claude-sonnet-4-5":          "claude-sonnet-4",
		"claude-sonnet-4-20250514":   "claude-sonnet-4",
		"claude-opus-4-5":            "claude-opus-4",
		"claude-haiku-4-5":           "claude-haiku-4",
		"gpt-4.1":                    "gpt-4.1",
		"gemini-2.5-pro":             "gemini-2.5-pro",
		"claude-sonnet-3-7-20250101": "claude-sonnet-3-7-20250101",
	}
	for input, expected := range cases {
		got := normalizeClaudeModel(input)
		if got != expected {
			t.Fatalf("normalizeClaudeModel(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestCopilotRequestPath(t *testing.T) {
	url := requestURLByMode("https://api.githubcopilot.com", relayconstant.RelayModeChatCompletions)
	if url != "https://api.githubcopilot.com/chat/completions" {
		t.Fatalf("chat url got %q", url)
	}

	url = requestURLByMode("https://api.githubcopilot.com", relayconstant.RelayModeEmbeddings)
	if url != "https://api.githubcopilot.com/embeddings" {
		t.Fatalf("embedding url got %q", url)
	}
}
