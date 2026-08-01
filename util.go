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
