// Package parser provides types and functions for parsing Codex CLI session
// JSONL files.
package parser

import (
	"encoding/json"
	"fmt"
)

// Event is the top-level envelope for every line in a Codex session JSONL file.
type Event struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// ---------------------------------------------------------------------------
// Subtype extraction
// ---------------------------------------------------------------------------

// subtypeEnvelope is used internally to peek at the "subtype" field of a
// payload without fully deserializing it.
type subtypeEnvelope struct {
	Subtype string `json:"subtype"`
}

// EventMsgType returns the subtype of an event_msg event. It returns an error
// if the event is not of type "event_msg" or the payload cannot be decoded.
func (e *Event) EventMsgType() (string, error) {
	if e.Type != "event_msg" {
		return "", fmt.Errorf("EventMsgType called on event of type %q, want \"event_msg\"", e.Type)
	}
	var env subtypeEnvelope
	if err := json.Unmarshal(e.Payload, &env); err != nil {
		return "", fmt.Errorf("decoding event_msg subtype: %w", err)
	}
	return env.Subtype, nil
}

// ---------------------------------------------------------------------------
// SessionMeta
// ---------------------------------------------------------------------------

// SessionMeta represents the session_meta event payload.
type SessionMeta struct {
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	CWD           string `json:"cwd"`
	Originator    string `json:"originator"`
	CLIVersion    string `json:"cli_version"`
	Source        string `json:"source"`
	ModelProvider string `json:"model_provider"`
}

// AsSessionMeta deserializes the payload as a SessionMeta.
func (e *Event) AsSessionMeta() (*SessionMeta, error) {
	var m SessionMeta
	if err := json.Unmarshal(e.Payload, &m); err != nil {
		return nil, fmt.Errorf("decoding session_meta payload: %w", err)
	}
	return &m, nil
}

// ---------------------------------------------------------------------------
// TurnContext
// ---------------------------------------------------------------------------

// TurnContext represents the turn_context event payload.
type TurnContext struct {
	TurnID            string            `json:"turn_id"`
	CWD               string            `json:"cwd"`
	Model             string            `json:"model"`
	Personality       string            `json:"personality"`
	ApprovalPolicy    string            `json:"approval_policy"`
	SandboxPolicy     SandboxPolicy     `json:"sandbox_policy"`
	CollaborationMode CollaborationMode `json:"collaboration_mode"`
}

// SandboxPolicy describes the sandboxing configuration.
type SandboxPolicy struct {
	Type string `json:"type"`
}

// CollaborationMode describes multi-model collaboration settings.
type CollaborationMode struct {
	Mode     string                    `json:"mode"`
	Settings CollaborationModeSettings `json:"settings"`
}

// CollaborationModeSettings contains the detailed model settings for
// collaboration mode.
type CollaborationModeSettings struct {
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort"`
}

// AsTurnContext deserializes the payload as a TurnContext.
func (e *Event) AsTurnContext() (*TurnContext, error) {
	var tc TurnContext
	if err := json.Unmarshal(e.Payload, &tc); err != nil {
		return nil, fmt.Errorf("decoding turn_context payload: %w", err)
	}
	return &tc, nil
}

// ---------------------------------------------------------------------------
// TokenCount (event_msg with subtype "token_count")
// ---------------------------------------------------------------------------

// TokenCount represents the payload of an event_msg with subtype
// "token_count".
type TokenCount struct {
	Subtype    string      `json:"subtype"`
	Info       TokenInfo   `json:"info"`
	RateLimits *RateLimits `json:"rate_limits,omitempty"`
}

// TokenInfo holds aggregated and per-turn token usage.
type TokenInfo struct {
	TotalTokenUsage    TokenUsage `json:"total_token_usage"`
	LastTokenUsage     TokenUsage `json:"last_token_usage"`
	ModelContextWindow int        `json:"model_context_window"`
}

// TokenUsage is the breakdown of token counts for a single measurement.
type TokenUsage struct {
	InputTokens           int `json:"input_tokens"`
	CachedInputTokens     int `json:"cached_input_tokens"`
	OutputTokens          int `json:"output_tokens"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens"`
	TotalTokens           int `json:"total_tokens"`
}

// RateLimits contains primary and secondary rate limit information.
type RateLimits struct {
	Primary   RateLimit `json:"primary"`
	Secondary RateLimit `json:"secondary"`
}

// RateLimit describes the usage within a single rate-limit window.
type RateLimit struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowMinutes int     `json:"window_minutes"`
	ResetsAt      int64   `json:"resets_at"`
}

// AsTokenCount deserializes the payload as a TokenCount. The caller should
// first verify that EventMsgType() == "token_count".
func (e *Event) AsTokenCount() (*TokenCount, error) {
	var tc TokenCount
	if err := json.Unmarshal(e.Payload, &tc); err != nil {
		return nil, fmt.Errorf("decoding token_count payload: %w", err)
	}
	return &tc, nil
}

// ---------------------------------------------------------------------------
// FunctionCall (response_item with subtype "function_call")
// ---------------------------------------------------------------------------

// FunctionCall represents the payload of a response_item with subtype
// "function_call".
type FunctionCall struct {
	Subtype   string `json:"subtype"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	CallID    string `json:"call_id"`
}

// AsFunctionCall deserializes the payload as a FunctionCall.
func (e *Event) AsFunctionCall() (*FunctionCall, error) {
	var fc FunctionCall
	if err := json.Unmarshal(e.Payload, &fc); err != nil {
		return nil, fmt.Errorf("decoding function_call payload: %w", err)
	}
	return &fc, nil
}
