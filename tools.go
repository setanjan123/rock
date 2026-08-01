package main

import (
	"encoding/json"
	"errors"
	"os"
)

func call_tool(tool_call ToolCall) (string, error) {
	tool_name := tool_call.Function.Name
	tool_args := tool_call.Function.Arguments
	switch tool_name {
	case "current_directory":
		return current_directory()
	case "list_directory":
		result, err := list_directory(tool_args)
		if err != nil {
			return "", err
		}

		encoded, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
	return "", errors.New("Invalid tool call")
}

func call_tools(tool_calls []ToolCall) []string {
	var tool_responses []string
	for _, tool_call := range tool_calls {
		tool_response, err := call_tool(tool_call)
		if err != nil {
			tool_response = err.Error()
		}
		tool_responses = append(tool_responses, tool_response)
	}
	return tool_responses
}

func get_tools() []ToolDefinition {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "current_directory",
				Description: "Returns the directory where the agent is running",
				Parameters: ToolParameters{
					Type:                 "object",
					Properties:           map[string]ToolProperty{},
					Required:             []string{},
					AdditionalProperties: false,
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "list_directory",
				Description: "Returns the contents of a directory",
				Parameters: ToolParameters{
					Type: "object",
					Properties: map[string]ToolProperty{
						"path": {
							Type:        "string",
							Description: "The directory path to list",
						},
					},
					Required:             []string{"path"},
					AdditionalProperties: false,
				},
			},
		},
	}
	return tools
}

func current_directory() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return dir, nil
}

func list_directory(path string) (ListDirectoryResult, error) {
	var args ListDirectoryArgs
	err := json.Unmarshal([]byte(path), &args)
	if err != nil {
		return ListDirectoryResult{}, err
	}

	if args.Path == "" {
		return ListDirectoryResult{}, errors.New("path is required")
	}

	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return ListDirectoryResult{}, err
	}

	result := ListDirectoryResult{
		Path:    args.Path,
		Entries: make([]string, 0, len(entries)),
	}
	for _, entry := range entries {
		result.Entries = append(result.Entries, entry.Name())
	}

	return result, nil
}
