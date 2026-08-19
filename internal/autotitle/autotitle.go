// Package autotitle turns one completed runner output into a short ephemeral
// label. It owns only the explicitly opted-in OpenAI API call; locating runner
// output and deciding when a session is eligible stay with the dashboard.
package autotitle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ez-gz/shepherd/internal/env"
	"github.com/ez-gz/shepherd/internal/format"
)

const (
	Model             = "gpt-5.6-luna"
	maxInputRunes     = 12_000
	maxTitleRunes     = 72
	maxErrorBodyBytes = 4 << 10
)

// Client is the small Responses API surface automatic titles need. Endpoint
// and HTTPClient are fields so tests never need the public network.
type Client struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

func FromEnvironment() Client {
	return Client{APIKey: env.Value(env.OpenAIAPIKey)}
}

func (c Client) Configured() bool { return strings.TrimSpace(c.APIKey) != "" }

func (c Client) Generate(ctx context.Context, completedOutput string) (string, error) {
	if !c.Configured() {
		return "", errors.New("automatic title requires OPENAI_API_KEY")
	}
	completedOutput = strings.TrimSpace(completedOutput)
	if completedOutput == "" {
		return "", errors.New("automatic title requires completed output")
	}
	runes := []rune(completedOutput)
	if len(runes) > maxInputRunes {
		completedOutput = string(runes[:maxInputRunes])
	}
	payload := map[string]any{
		"model": Model,
		"store": false,
		"input": []map[string]any{
			{"role": "developer", "content": "Name this coding-agent session in 2 to 6 concrete words. Return only the title, with no quotes, punctuation, or explanation."},
			{"role": "user", "content": completedOutput},
		},
		"reasoning":         map[string]any{"effort": "none"},
		"text":              map[string]any{"verbosity": "low"},
		"max_output_tokens": 32,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimSpace(c.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/responses"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+c.APIKey)
	request.Header.Set("Content-Type", "application/json")
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("generate automatic title: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		return "", fmt.Errorf("generate automatic title: OpenAI returned %s: %s",
			response.Status, format.OneLine(string(message)))
	}
	var result struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&result); err != nil {
		return "", fmt.Errorf("decode automatic title: %w", err)
	}
	for _, output := range result.Output {
		if output.Type != "message" {
			continue
		}
		for _, content := range output.Content {
			if content.Type != "output_text" {
				continue
			}
			title := strings.Trim(format.OneLine(content.Text), " \t\r\n\"'`")
			if title == "" {
				continue
			}
			runes := []rune(title)
			if len(runes) > maxTitleRunes {
				title = string(runes[:maxTitleRunes])
			}
			return title, nil
		}
	}
	return "", errors.New("automatic title response contained no text")
}
