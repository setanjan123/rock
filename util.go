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

func handle_response(response ChatCompletionResponse, messages *[]Message, is_toolcall_continue *bool) {
	var finish_reason = response.Choices[0].FinishReason
	switch finish_reason {
	case "tool_calls":
		var tool_calls = response.Choices[0].Message.ToolCalls
		tool_responses := call_tools(tool_calls)
		*messages = append(*messages, Message{Role: "assistant", ToolCalls: tool_calls})

		for index, tool_response := range tool_responses {
			*messages = append(*messages, Message{Role: "tool", ToolCallID: tool_calls[index].ID, Content: tool_response})
		}
		*is_toolcall_continue = true
	case "stop":
		*messages = append(*messages, Message{Role: "assistant", Content: response.Choices[0].Message.Content})
		fmt.Println()
		fmt.Println("Agent ›")
		fmt.Println(response.Choices[0].Message.Content)
		*is_toolcall_continue = false
	}
}
