package autotitle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateUsesTheSinglePurposeLunaResponsesContract(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("request = %s authorization %q", request.Method, request.Header.Get("Authorization"))
		}
		var payload struct {
			Model           string           `json:"model"`
			Store           *bool            `json:"store"`
			Input           []map[string]any `json:"input"`
			Reasoning       map[string]any   `json:"reasoning"`
			Text            map[string]any   `json:"text"`
			MaxOutputTokens int              `json:"max_output_tokens"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Model != Model || payload.Store == nil || *payload.Store || payload.Reasoning["effort"] != "none" ||
			payload.Text["verbosity"] != "low" || payload.MaxOutputTokens != 32 {
			t.Fatalf("payload = %#v", payload)
		}
		if len(payload.Input) != 2 || payload.Input[1]["role"] != "user" ||
			payload.Input[1]["content"] != "Implemented bounded native status" {
			t.Fatalf("input = %#v", payload.Input)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"  \"Native Runner Status\"  "}]}]}`))
	}))
	defer server.Close()

	title, err := (Client{APIKey: "test-key", Endpoint: server.URL, HTTPClient: server.Client()}).Generate(
		context.Background(), "Implemented bounded native status")
	if err != nil {
		t.Fatal(err)
	}
	if title != "Native Runner Status" || requests != 1 {
		t.Fatalf("title = %q, requests = %d", title, requests)
	}
}

func TestGenerateWithoutAKeyCannotReachHTTP(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()

	_, err := (Client{Endpoint: server.URL, HTTPClient: server.Client()}).Generate(context.Background(), "completed")
	if err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("made %d requests without a key", requests)
	}
}

func TestGeneratedTitleIsBoundedAndSingleLine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"` +
			strings.Repeat("界", maxTitleRunes+20) + `\nignored"}]}]}`))
	}))
	defer server.Close()

	title, err := (Client{APIKey: "key", Endpoint: server.URL, HTTPClient: server.Client()}).Generate(context.Background(), "completed")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(title, "\n") || len([]rune(title)) != maxTitleRunes {
		t.Fatalf("title = %q (%d runes)", title, len([]rune(title)))
	}
}
