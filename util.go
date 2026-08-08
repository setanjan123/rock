package main

import (
	"fmt"
	"os"
	"strings"
)

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
