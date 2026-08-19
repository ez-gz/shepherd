package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ez-gz/shepherd/internal/format"
	"github.com/ez-gz/shepherd/internal/shepherd"
)

// CompletedOutputRequest names a launch whose first completed assistant turn
// may be used as automatic-title input. SessionID is the native conversation
// id when known; Codex may leave it empty and be matched from the immutable
// launch facts instead.
type CompletedOutputRequest struct {
	Runner    shepherd.Backend
	SessionID string
	Root      string
	StartedAt time.Time
	Prompt    string
}

// FirstCompletedOutput returns the first runner-declared completed assistant
// output. A transcript that exists but has not completed a turn is found=false,
// not an error. Terminal contents and timing heuristics are never consulted.
func (r Reader) FirstCompletedOutput(request CompletedOutputRequest) (text string, found bool, err error) {
	switch request.Runner {
	case shepherd.BackendClaude:
		path, err := r.locateClaude(strings.TrimSpace(request.SessionID), request.Root)
		if err != nil || path == "" {
			return "", false, err
		}
		return firstClaudeCompletedOutput(path, request.StartedAt)
	case shepherd.BackendCodex:
		conversation, err := r.FindConversation(ConversationRequest{
			Runner: request.Runner, Root: request.Root, StartedAt: request.StartedAt, Prompt: request.Prompt,
		})
		if err != nil {
			if errors.Is(err, ErrConversationNotFound) {
				return "", false, nil
			}
			return "", false, err
		}
		return firstCodexCompletedOutput(conversation.Path, request.StartedAt)
	default:
		return "", false, nil
	}
}

func firstClaudeCompletedOutput(path string, startedAt time.Time) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("open transcript %s: %w", format.CompactPath(path), err)
	}
	defer file.Close()

	limited := &io.LimitedReader{R: file, N: maxTranscriptBytes}
	reader := bufio.NewReaderSize(limited, 64<<10)
	var parts []string
	for {
		line, _, readErr := readBoundedLine(reader, maxLineBytes)
		if len(bytes.TrimSpace(line)) > 0 {
			if record, ok := decodeRecord(line); ok {
				currentLaunch := startedAt.IsZero() || (!record.Timestamp.IsZero() && !record.Timestamp.Before(startedAt))
				if currentLaunch {
					role, texts, _, skip := interpret(record)
					switch {
					case !skip && role == RoleUser:
						parts = nil
					case !skip && role == RoleAssistant:
						parts = append(parts, texts...)
						if reason := record.Message.StopReason; reason != "" && reason != "tool_use" {
							text := strings.TrimSpace(strings.Join(parts, "\n"))
							return text, text != "", nil
						}
					}
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return "", false, nil
			}
			return "", false, fmt.Errorf("read transcript %s: %w", format.CompactPath(path), readErr)
		}
	}
}

func firstCodexCompletedOutput(path string, startedAt time.Time) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("open transcript %s: %w", format.CompactPath(path), err)
	}
	defer file.Close()

	limited := &io.LimitedReader{R: file, N: maxTranscriptBytes}
	reader := bufio.NewReaderSize(limited, 64<<10)
	for {
		line, _, readErr := readBoundedLine(reader, maxLineBytes)
		if len(bytes.TrimSpace(line)) > 0 {
			var record struct {
				Timestamp time.Time `json:"timestamp"`
				Type      string    `json:"type"`
				Payload   struct {
					Type             string `json:"type"`
					LastAgentMessage string `json:"last_agent_message"`
				} `json:"payload"`
			}
			if json.Unmarshal(line, &record) == nil && record.Type == "event_msg" && record.Payload.Type == "task_complete" &&
				(startedAt.IsZero() || (!record.Timestamp.IsZero() && !record.Timestamp.Before(startedAt))) {
				text := strings.TrimSpace(record.Payload.LastAgentMessage)
				return text, text != "", nil
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return "", false, nil
			}
			return "", false, fmt.Errorf("read transcript %s: %w", format.CompactPath(path), readErr)
		}
	}
}
