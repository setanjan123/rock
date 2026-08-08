package main

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultContextLimit = 32768 // fallback when CONTEXT_LIMIT is not set
	compactionThreshold = 0.8   // compact when estimated tokens exceed 80% of the limit
	keepRecentMessages  = 10    // messages to keep at the tail of the conversation
	tokenCharsRatio     = 4     // 1 token ≈ 4 characters (heuristic)
)

// get_context_limit returns the token limit from the CONTEXT_LIMIT env var,
// falling back to defaultContextLimit when it is unset or invalid.
func get_context_limit() int {
	contextLimit := os.Getenv("CONTEXT_LIMIT")
	if contextLimit != "" {
		num, err := strconv.Atoi(contextLimit)
		if err == nil {
			return num
		}
	}
	fmt.Println("Error reading context limit. Going with default value")
	return defaultContextLimit
}

// estimate_tokens returns a rough token count for a single message
// (len(Content) / tokenCharsRatio), plus any ToolCall function name and
// arguments JSON.
func estimate_tokens(msg Message) int {
	tokens := len(msg.Content) / tokenCharsRatio
	for _, tc := range msg.ToolCalls {
		tokens += len(tc.Function.Name)/tokenCharsRatio + len(tc.Function.Arguments)/tokenCharsRatio
	}
	return tokens
}

// total_estimated_tokens sums estimate_tokens across all messages.
func total_estimated_tokens(messages []Message) int {
	estimation := 0
	for _, msg := range messages {
		estimation += estimate_tokens(msg)
	}
	return estimation
}

// needs_compaction reports whether the estimated context usage is more than
// compactionThreshold of the limit.
func needs_compaction(contextUsage int, limit int) bool {
	return float64(contextUsage)/float64(limit) > compactionThreshold
}

// track_context_usage adds a message's token estimate to the running count
// that is maintained alongside the conversation history.
func track_context_usage(contextUsage *int, msg Message) {
	*contextUsage += estimate_tokens(msg)
}

func get_context_usage(contextUsage int) string {
	context_limit := get_context_limit()
	return fmt.Sprintf("%d/%d", contextUsage, context_limit)
}

// compact_context summarizes older messages via a dedicated no-tools API call
// and replaces them with the summary, preserving the system prompt and recent
// messages. Returns an error if the summarization request fails.
func compact_context(messages *[]Message, apiKey *string, baseURL *string, model *string) error {

	// 1. Build summarization request from messages (exclude system prompt[0])
	summary_request := build_summary_request(*messages)

	// 2. Call summarization API
	summary_response, err := call_ai(&summary_request, apiKey, baseURL, model, nil)
	if err != nil {
		return err
	}

	// 3. Replace older messages with summary
	replace_with_summary(messages, summary_response.Choices[0].Message.Content)

	return nil

}

// build_summary_request returns a two-message request (summarizer system
// prompt + the conversation to summarize) for the dedicated summarization call.
func build_summary_request(messages []Message) []Message {
	result := append([]Message{{Role: "system", Content: "You are a conversation summarizer. Produce a concise summary of the conversation between a user and an AI coding assistant. Include: key facts, decisions made, files examined or modified, errors encountered, current task, and any open questions."},
		{Role: "user", Content: "Summarize the following conversation. Be thorough but concise. Output only the summary, no preamble"}}, messages[1:]...)
	return result
}

// replace_with_summary rewrites the history in place to
// [system prompt, summary, last keepRecentMessages messages].
func replace_with_summary(messages *[]Message, summary string) {
	systemprompt := (*messages)[0]
	lasttenmessages := []Message{}
	if len(*messages) > keepRecentMessages+1 {
		lasttenmessages = (*messages)[len(*messages)-keepRecentMessages:]
	} else {
		lasttenmessages = (*messages)[1:]
	}
	*messages = append([]Message{systemprompt, {Role: "system", Content: "[Conversation summary so far]\n" + summary}}, lasttenmessages...)
}
