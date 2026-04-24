package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	githubDeviceCodeURL = "https://github.com/login/device/code"
	githubAccessTokenURL = "https://github.com/login/oauth/access_token"
	githubCopilotVSCodeClientID = "Iv1.b507a08c87ecfe98"
)

var (
	ErrGitHubOAuthPending  = fmt.Errorf("github oauth authorization pending")
	ErrGitHubOAuthSlowDown = fmt.Errorf("github oauth slow down")
)

type GitHubDeviceCodeResult struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type GitHubAccessTokenResult struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func githubDeviceClientID() string {
	if v := strings.TrimSpace(os.Getenv("GITHUB_DEVICE_CLIENT_ID")); v != "" {
		return v
	}
	// Align with VS Code Copilot official OAuth client for github.com.
	return githubCopilotVSCodeClientID
}

func githubDeviceScope() string {
	scope := strings.TrimSpace(os.Getenv("GITHUB_DEVICE_SCOPE"))
	if scope == "" {
		return "read:user"
	}
	return scope
}

func StartGitHubDeviceFlow(ctx context.Context) (*GitHubDeviceCodeResult, error) {
	clientID := githubDeviceClientID()
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", githubDeviceScope())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubDeviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out GitHubDeviceCodeResult
	if err = common.DecodeJson(resp.Body, &out); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github device code failed: status=%d", resp.StatusCode)
	}
	if out.DeviceCode == "" || out.UserCode == "" || out.VerificationURI == "" {
		return nil, fmt.Errorf("github device code response missing fields")
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	if out.ExpiresIn <= 0 {
		out.ExpiresIn = int((15 * time.Minute).Seconds())
	}
	return &out, nil
}

func PollGitHubDeviceAccessToken(ctx context.Context, deviceCode string) (*GitHubAccessTokenResult, error) {
	clientID := githubDeviceClientID()
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("device_code", strings.TrimSpace(deviceCode))
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubAccessTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload map[string]any
	if err = common.DecodeJson(resp.Body, &payload); err != nil {
		return nil, err
	}

	if errCode := common.Interface2String(payload["error"]); errCode != "" {
		switch errCode {
		case "authorization_pending":
			return nil, ErrGitHubOAuthPending
		case "slow_down":
			return nil, ErrGitHubOAuthSlowDown
		default:
			return nil, fmt.Errorf("github oauth failed: %s", errCode)
		}
	}

	result := &GitHubAccessTokenResult{
		AccessToken: common.Interface2String(payload["access_token"]),
		TokenType:   common.Interface2String(payload["token_type"]),
		Scope:       common.Interface2String(payload["scope"]),
	}
	if result.AccessToken == "" {
		return nil, fmt.Errorf("github oauth response missing access_token")
	}
	return result, nil
}
