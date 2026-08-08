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
// estimates to 10 tokens, an empty message to 0. Tool calls are also
// counted from their Function.Name and Function.Arguments fields.
func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		name string
		msg  Message
		want int
	}{
		{"empty content", message("user", ""), 0},
		{"short content", message("user", "abcd"), 1},
		{"exact multiple of four", message("user", strings.Repeat("a", 40)), 10},
		{"rounded down", message("user", strings.Repeat("a", 41)), 10},
		{
			"tool call without content",
			Message{
				Role:    "assistant",
				Content: "",
				ToolCalls: []ToolCall{{
					Function: Function{Name: "read_file", Arguments: `{"path":"/foo/bar.txt"}`},
				}},
			},
				7, // "read_file"(9/4=2) + `{"path":"/foo/bar.txt"}`(23/4=5) = 7
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := estimate_tokens(tc.msg)
			if got != tc.want {
				t.Fatalf("estimate_tokens(%+v) = %d, want %d", tc.msg, got, tc.want)
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

// TestNeedsCompaction verifies the 80% threshold boundary directly against
// the maintained contextUsage counter (no rescan of messages).
func TestNeedsCompaction(t *testing.T) {
	const limit = 100

	if needs_compaction(80, limit) {
		t.Fatalf("needs_compaction = true at exactly 80 tokens, want false (threshold is strictly >)")
	}

	if !needs_compaction(81, limit) {
		t.Fatalf("needs_compaction = false at 81 tokens, want true")
	}

	if needs_compaction(0, limit) {
		t.Fatalf("needs_compaction = true for an empty context, want false")
	}
}

// TestTrackContextUsage verifies that the running counter stays in sync with
// the conversation by mirroring total_estimated_tokens across appends.
func TestTrackContextUsage(t *testing.T) {
	messages := []Message{message("system", strings.Repeat("a", 40))} // 10 tokens
	usage := total_estimated_tokens(messages)

	appends := []Message{
		message("user", strings.Repeat("b", 40)), // 10 tokens
		{ // tool-call message: 7 tokens
			Role: "assistant",
			ToolCalls: []ToolCall{{
				Function: Function{Name: "read_file", Arguments: `{"path":"/foo/bar.txt"}`},
			}},
		},
		message("tool", strings.Repeat("c", 40)), // 10 tokens
		message("assistant", strings.Repeat("d", 40)), // 10 tokens
	}
	for _, msg := range appends {
		messages = append(messages, msg)
		track_context_usage(&usage, msg)
	}

	if want := total_estimated_tokens(messages); usage != want {
		t.Fatalf("tracked usage = %d, want %d (must match total_estimated_tokens)", usage, want)
	}
}

// TestHandleResponseTracksUsage verifies the real wiring: handle_response must
// increment the running counter for every message it appends, in both the
// tool_calls branch (assistant tool-call message + tool result) and the stop
// branch (assistant text message).
func TestHandleResponseTracksUsage(t *testing.T) {
	// --- stop branch ---
	stopMessages := []Message{message("system", "prompt")}
	stopUsage := total_estimated_tokens(stopMessages)
	stopResponse := ChatCompletionResponse{
		Choices: []Choice{{FinishReason: "stop", Message: message("assistant", strings.Repeat("a", 40))}}, // 10 tokens
	}
	isContinue := true
	handle_response(stopResponse, &stopMessages, &isContinue, &stopUsage)
	if want := total_estimated_tokens(stopMessages); stopUsage != want {
		t.Fatalf("stop branch: tracked usage = %d, want %d (must equal total_estimated_tokens)", stopUsage, want)
	}
	if isContinue {
		t.Fatalf("stop branch: is_toolcall_continue = true, want false")
	}
	if stopMessages[len(stopMessages)-1].Role != "assistant" {
		t.Fatalf("stop branch: last message role = %q, want assistant", stopMessages[len(stopMessages)-1].Role)
	}

	// --- tool_calls branch (executes current_directory, no filesystem writes) ---
	toolMessages := []Message{message("system", "prompt")}
	toolUsage := total_estimated_tokens(toolMessages)
	toolResponse := ChatCompletionResponse{
		Choices: []Choice{{
			FinishReason: "tool_calls",
			Message: Message{
				Role: "assistant",
				ToolCalls: []ToolCall{{
					ID:       "call_1",
					Type:     "function",
					Function: Function{Name: "current_directory", Arguments: "{}"},
				}},
			},
		}},
	}
	isContinue = false
	handle_response(toolResponse, &toolMessages, &isContinue, &toolUsage)
	if want := total_estimated_tokens(toolMessages); toolUsage != want {
		t.Fatalf("tool_calls branch: tracked usage = %d, want %d (must equal total_estimated_tokens)", toolUsage, want)
	}
	if !isContinue {
		t.Fatalf("tool_calls branch: is_toolcall_continue = false, want true")
	}
	// The tool_calls branch appends the assistant tool-call message plus one tool result.
	if want := 3; len(toolMessages) != want {
		t.Fatalf("tool_calls branch: got %d messages, want %d (system, assistant, tool)", len(toolMessages), want)
	}
	if toolMessages[len(toolMessages)-1].Role != "tool" {
		t.Fatalf("tool_calls branch: last message role = %q, want tool", toolMessages[len(toolMessages)-1].Role)
	}
}

// TestTrackContextUsageAfterCompaction verifies that recomputing the counter
// from messages after replace_with_summary keeps it in sync with the new
// (much smaller) history.
func TestTrackContextUsageAfterCompaction(t *testing.T) {
	messages := []Message{message("system", "prompt")}
	for i := 1; i <= 20; i++ {
		messages = append(messages, message("user", strings.Repeat("x", 400))) // 100 tokens each
	}
	usage := total_estimated_tokens(messages)

	replace_with_summary(&messages, "compacted summary")
	usage = total_estimated_tokens(messages)

	if want := total_estimated_tokens(messages); usage != want {
		t.Fatalf("recomputed usage = %d, want %d", usage, want)
	}
	// After compaction the counter must reflect the shrunken history, not the
	// pre-compaction 2010-token count.
	if usage >= 2000 {
		t.Fatalf("recomputed usage = %d, want a small post-compaction count", usage)
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
