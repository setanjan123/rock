package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func load_config() (string, string, string, error) {
	// Load .env if present; exported env vars take precedence either way.
	_ = godotenv.Load()

	apiKey := os.Getenv("API_KEY")
	baseURL := os.Getenv("API_BASE_URL")
	model := os.Getenv("MODEL")

	for _, name := range []string{"API_KEY", "API_BASE_URL", "MODEL"} {
		if os.Getenv(name) == "" {
			return "", "", "", errors.New("missing required env var " + name + " (set it or add it to a .env file)")
		}
	}

	return apiKey, baseURL, model, nil
}


func verbose_tools() bool {
	v := strings.TrimSpace(os.Getenv("VERBOSE_TOOLS"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func print_tool_calls(tool_calls []ToolCall) {
	if !verbose_tools() {
		return
	}
	fmt.Println(promptSeparator)
	fmt.Println("Agent called tools:")
	for _, tc := range tool_calls {
		fmt.Printf("  - %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
	}
}

func read_system_prompt(messages *[]Message) {
	// Read the entire file into memory
	content, err := os.ReadFile("system.md")
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Error reading system prompt", err.Error())
		return
	}
	systemprompt := strings.TrimSpace(string(content))

	if systemprompt != "" {
		*messages = append(*messages, Message{Role: "system", Content: systemprompt})
	}

}

func handle_response(response ChatCompletionResponse, messages *[]Message, is_toolcall_continue *bool, contextUsage *int) {
	var finish_reason = response.Choices[0].FinishReason
	switch finish_reason {
	case "tool_calls":
		var tool_calls = response.Choices[0].Message.ToolCalls
		tool_responses := call_tools(tool_calls)
		assistantMsg := Message{Role: "assistant", ToolCalls: tool_calls}
		*messages = append(*messages, assistantMsg)
		track_context_usage(contextUsage, assistantMsg)
		print_tool_calls(tool_calls)

		for index, tool_response := range tool_responses {
			toolMsg := Message{Role: "tool", ToolCallID: tool_calls[index].ID, Content: tool_response}
			*messages = append(*messages, toolMsg)
			track_context_usage(contextUsage, toolMsg)
		}
		*is_toolcall_continue = true
	case "stop":
		assistantMsg := Message{Role: "assistant", Content: response.Choices[0].Message.Content}
		*messages = append(*messages, assistantMsg)
		track_context_usage(contextUsage, assistantMsg)
		fmt.Println()
		fmt.Println("Agent ›")
		fmt.Println(response.Choices[0].Message.Content)
		*is_toolcall_continue = false
	}
}
