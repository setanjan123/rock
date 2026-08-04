package main

import (
	"strings"
	"testing"
)

// message builds a simple text message with the given role and content.
func message(role, content string) Message {
	return Message{Role: role, Content: content}
}

// TestEstimateTokens verifies the chars/4 heuristic: a 40-char message
// estimates to 10 tokens, an empty message to 0.
func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{"empty content", "", 0},
		{"short content", "abcd", 1},
		{"exact multiple of four", strings.Repeat("a", 40), 10},
		{"rounded down", strings.Repeat("a", 41), 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := estimate_tokens(message("user", tc.content))
			if got != tc.want {
				t.Fatalf("estimate_tokens(%q) = %d, want %d", tc.content, got, tc.want)
			}
		})
	}
}

// TestTotalEstimatedTokens verifies the sum across multiple messages.
func TestTotalEstimatedTokens(t *testing.T) {
	messages := []Message{
		message("system", strings.Repeat("a", 40)), // 10 tokens
		message("user", strings.Repeat("b", 40)),   // 10 tokens
		message("assistant", strings.Repeat("c", 40)), // 10 tokens
	}
	got := total_estimated_tokens(messages)
	if got != 30 {
		t.Fatalf("total_estimated_tokens = %d, want 30", got)
	}
}

// TestNeedsCompaction verifies the 80% threshold boundary.
// With a limit of 100, compaction triggers only above 80 estimated tokens.
func TestNeedsCompaction(t *testing.T) {
	const limit = 100

	below := []Message{message("user", strings.Repeat("a", 320))} // 80 tokens, not above
	if needs_compaction(below, limit) {
		t.Fatalf("needs_compaction = true at exactly 80 tokens, want false (threshold is strictly >)")
	}

	above := []Message{message("user", strings.Repeat("a", 324))} // 81 tokens
	if !needs_compaction(above, limit) {
		t.Fatalf("needs_compaction = false at 81 tokens, want true")
	}

	empty := []Message{}
	if needs_compaction(empty, limit) {
		t.Fatalf("needs_compaction = true for empty conversation, want false")
	}
}

// TestGetContextLimit verifies env-var parsing and fallback behavior.
func TestGetContextLimit(t *testing.T) {
	t.Run("unset falls back to default", func(t *testing.T) {
		t.Setenv("CONTEXT_LIMIT", "")
		if got := get_context_limit(); got != defaultContextLimit {
			t.Fatalf("get_context_limit() = %d, want default %d", got, defaultContextLimit)
		}
	})

	t.Run("valid value is used", func(t *testing.T) {
		t.Setenv("CONTEXT_LIMIT", "4096")
		if got := get_context_limit(); got != 4096 {
			t.Fatalf("get_context_limit() = %d, want 4096", got)
		}
	})

	t.Run("invalid value falls back to default", func(t *testing.T) {
		t.Setenv("CONTEXT_LIMIT", "not-a-number")
		if got := get_context_limit(); got != defaultContextLimit {
			t.Fatalf("get_context_limit() = %d, want default %d", got, defaultContextLimit)
		}
	})
}

// TestBuildSummaryRequest verifies the summarization request shape:
// a summarizer system prompt, a user instruction, then all messages
// except the original system prompt.
func TestBuildSummaryRequest(t *testing.T) {
	original := []Message{
		message("system", "You are a coding assistant."),
		message("user", "first question"),
		message("assistant", "first answer"),
	}
	request := build_summary_request(original)

	if len(request) != 4 {
		t.Fatalf("build_summary_request returned %d messages, want 4", len(request))
	}
	if request[0].Role != "system" {
		t.Fatalf("request[0].Role = %q, want system", request[0].Role)
	}
	if !strings.Contains(request[0].Content, "conversation summarizer") {
		t.Fatalf("request[0].Content missing summarizer instructions: %q", request[0].Content)
	}
	if request[1].Role != "user" {
		t.Fatalf("request[1].Role = %q, want user", request[1].Role)
	}
	// The remaining messages are the conversation minus the system prompt.
	if request[2].Content != "first question" || request[3].Content != "first answer" {
		t.Fatalf("conversation messages not carried through correctly: %+v", request[2:])
	}
}

// TestReplaceWithSummary verifies the full replacement:
// [system prompt, summary, last keepRecentMessages messages].
func TestReplaceWithSummary(t *testing.T) {
	messages := []Message{message("system", "prompt")}
	for i := 1; i <= 20; i++ {
		messages = append(messages, message("user", string(rune('a'+i))))
	}

	replace_with_summary(&messages, "compacted summary")

	if len(messages) != keepRecentMessages+2 { // system + summary + 10 recent
		t.Fatalf("after replacement got %d messages, want %d", len(messages), keepRecentMessages+2)
	}
	if messages[0].Content != "prompt" {
		t.Fatalf("system prompt was not preserved: %q", messages[0].Content)
	}
	if messages[1].Role != "system" || !strings.Contains(messages[1].Content, "compacted summary") {
		t.Fatalf("summary message not inserted at index 1: %+v", messages[1])
	}
	// The last message must be the most recent conversation message.
	if messages[len(messages)-1].Content != "u" { // 'a' + 20
		t.Fatalf("most recent message not preserved, got %q", messages[len(messages)-1].Content)
	}
}

// TestReplaceWithSummaryFewMessages verifies that a conversation with
// fewer than keepRecentMessages messages keeps everything intact.
func TestReplaceWithSummaryFewMessages(t *testing.T) {
	messages := []Message{
		message("system", "prompt"),
		message("user", "hi"),
		message("assistant", "hello"),
	}

	replace_with_summary(&messages, "compacted summary")

	if len(messages) != 4 { // system + summary + both original messages
		t.Fatalf("after replacement got %d messages, want 4", len(messages))
	}
	if messages[0].Content != "prompt" {
		t.Fatalf("system prompt was not preserved: %q", messages[0].Content)
	}
	if !strings.Contains(messages[1].Content, "compacted summary") {
		t.Fatalf("summary message missing: %+v", messages[1])
	}
	if messages[2].Content != "hi" || messages[3].Content != "hello" {
		t.Fatalf("original messages not preserved: %+v", messages[2:])
	}
}
