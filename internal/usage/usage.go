// Package usage fetches real-time rate limit data from the OpenAI WHAM API,
// the same endpoint that Codex CLI's /status command uses.
package usage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// authFile represents the structure of ~/.codex/auth.json.
type authFile struct {
	AuthMode string     `json:"auth_mode"`
	Tokens   authTokens `json:"tokens"`
}

type authTokens struct {
	AccessToken string `json:"access_token"`
	AccountID   string `json:"account_id"`
}

// Response is the parsed WHAM /usage API response.
type Response struct {
	PlanType  string           `json:"plan_type"`
	RateLimit *RateLimitStatus `json:"rate_limit"`
}

// RateLimitStatus contains the rate limit details.
type RateLimitStatus struct {
	Allowed      bool            `json:"allowed"`
	LimitReached bool            `json:"limit_reached"`
	Primary      *WindowSnapshot `json:"primary_window"`
	Secondary    *WindowSnapshot `json:"secondary_window"`
}

// WindowSnapshot represents a single rate limit window.
type WindowSnapshot struct {
	UsedPercent       int `json:"used_percent"`
	LimitWindowSecs   int `json:"limit_window_seconds"`
	ResetAfterSeconds int `json:"reset_after_seconds"`
	ResetAt           int `json:"reset_at"`
}

// Fetch reads auth credentials from ~/.codex/auth.json and queries the
// WHAM /usage API for real-time rate limit data.
func Fetch() (*Response, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find home directory: %w", err)
	}

	authPath := filepath.Join(home, ".codex", "auth.json")
	auth, err := readAuth(authPath)
	if err != nil {
		return nil, fmt.Errorf("reading auth: %w", err)
	}

	if auth.Tokens.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in auth.json")
	}

	return fetchUsage(auth)
}

func readAuth(path string) (*authFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var af authFile
	if err := json.Unmarshal(data, &af); err != nil {
		return nil, err
	}
	return &af, nil
}

func fetchUsage(auth *authFile) (*Response, error) {
	url := "https://chatgpt.com/backend-api/wham/usage"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+auth.Tokens.AccessToken)
	req.Header.Set("User-Agent", "codex-cli")
	if auth.Tokens.AccountID != "" {
		req.Header.Set("Chatgpt-Account-Id", auth.Tokens.AccountID)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}
