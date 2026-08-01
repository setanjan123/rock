package main

import (
	"fmt"
	"os"
	"strings"
)

func read_system_prompt(messages *[]map[string]any) {
	// Read the entire file into memory
	content, err := os.ReadFile("system.md")
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Error reading system prompt", err.Error())
		return
	}
	systemprompt := strings.TrimSpace(string(content))

	if systemprompt != "" {
		message := make(map[string]any)
		message["role"] = "system"
		message["content"] = systemprompt
		*messages = append(*messages, message)
	}

}

func handle_response(response ChatCompletionResponse, messages *[]map[string]any, is_toolcall_continue *bool) {
	var finish_reason = response.Choices[0].FinishReason
	message := make(map[string]any)
	message["role"] = "assistant"
	switch finish_reason {
	case "tool_calls":
		var tool_calls = response.Choices[0].Message.ToolCalls
		tool_responses := call_tools(tool_calls)
		message["content"] = nil
		message["tool_calls"] = tool_calls
		*messages = append(*messages, message)

		for index, tool_response := range tool_responses {
			toolmessage := make(map[string]any)
			toolmessage["role"] = "tool"
			toolmessage["tool_call_id"] = tool_calls[index].ID
			toolmessage["content"] = tool_response
			*messages = append(*messages, toolmessage)
		}
		*is_toolcall_continue = true
	case "stop":
		message["content"] = response.Choices[0].Message.Content
		*messages = append(*messages, message)
		fmt.Println()
		fmt.Println("Agent ›")
		fmt.Println(message["content"])
		*is_toolcall_continue = false
	}
}
