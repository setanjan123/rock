package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"time"
)

func call_tool(index int, tool_call ToolCall, responses chan ToolCallResponse) {
	var response ToolCallResponse
	response.Index = index
	switch tool_call.Function.Name {
	case "current_directory":
		resp, err := current_directory()
		if err != nil {
			response.Error = err
		} else {
			response.Response = resp
		}
	case "list_directory":
		resp, err := list_directory(tool_call.Function.Arguments)
		if err != nil {
			response.Error = err
		} else {
			response.Response = resp
		}
	case "read_file":
		resp, err := read_file(tool_call.Function.Arguments)
		if err != nil {
			response.Error = err
		} else {
			response.Response = resp
		}
	case "write_file":
		resp, err := write_file(tool_call.Function.Arguments)
		if err != nil {
			response.Error = err
		} else {
			response.Response = resp
		}
	case "delete_file":
		resp, err := delete_file(tool_call.Function.Arguments)
		if err != nil {
			response.Error = err
		} else {
			response.Response = resp
		}
	case "exec_command":
		resp, err := exec_command(tool_call.Function.Arguments)
		response.Error = err
		response.Response = resp
	default:
		response.Error = errors.New("Invalid tool call")
	}
	responses <- response
}

func call_tools(tool_calls []ToolCall) []string {
	tool_call_responses := make(chan ToolCallResponse)
	for index, tool_call := range tool_calls {
		go call_tool(index, tool_call, tool_call_responses)
	}
	tool_responses := make([]string, len(tool_calls))
	for n := 0; n < len(tool_calls); n++ {
		tool_response := <-tool_call_responses
		if tool_response.Error != nil {
			result := tool_response.Error.Error()
			if tool_response.Response != "" {
				result += "\n" + tool_response.Response
			}
			tool_responses[tool_response.Index] = result
		} else {
			tool_responses[tool_response.Index] = tool_response.Response
		}
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
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "read_file",
				Description: "Returns the contents of a file",
				Parameters: ToolParameters{
					Type: "object",
					Properties: map[string]ToolProperty{
						"path": {
							Type:        "string",
							Description: "The path to the file",
						},
					},
					Required:             []string{"path"},
					AdditionalProperties: false,
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "write_file",
				Description: "Write a file",
				Parameters: ToolParameters{
					Type: "object",
					Properties: map[string]ToolProperty{
						"path": {
							Type:        "string",
							Description: "The path to the file",
						},
						"contents": {
							Type:        "string",
							Description: "The contents of the file",
						},
					},
					Required:             []string{"path"},
					AdditionalProperties: false,
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "delete_file",
				Description: "Deletes a file by path. This is the dedicated utility for safe removal.",
				Parameters: ToolParameters{
					Type: "object",
					Properties: map[string]ToolProperty{
						"path": {
							Type:        "string",
							Description: "The path to the file that needs to be deleted.",
						},
					},
					Required:             []string{"path"},
					AdditionalProperties: false,
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "exec_command",
				Description: "Execute a command (use this only when dedicated tools like delete_file are insufficient).",
				Parameters: ToolParameters{
					Type: "object",
					Properties: map[string]ToolProperty{
						"command": {
							Type:        "string",
							Description: "The executable to run, such as \"go\" or \"powershell\". Do not include arguments here.",
						},
						"args": {
							Type:        "array",
							Description: "Arguments passed to the executable separately, if any.",
							Items:       &ToolProperty{Type: "string"},
						},
					},
					Required:             []string{"command"},
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

func list_directory(path string) (string, error) {
	var args ListDirectoryArgs
	err := json.Unmarshal([]byte(path), &args)
	if err != nil {
		return "", err
	}

	if args.Path == "" {
		return "", errors.New("path is required")
	}

	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return "", err
	}

	result := ListDirectoryResult{
		Path:    args.Path,
		Entries: make([]string, 0, len(entries)),
	}
	for _, entry := range entries {
		result.Entries = append(result.Entries, entry.Name())
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func read_file(path string) (string, error) {
	var args ReadFileArgs
	err := json.Unmarshal([]byte(path), &args)
	if err != nil {
		return "", err
	}

	if args.Path == "" {
		return "", errors.New("path is required")
	}

	file, err := os.Open(args.Path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	var fileContent string
	for scanner.Scan() {
		// Read current line as a string
		line := scanner.Text()
		fileContent = fileContent + "\n" + line
	}

	// Check for any errors encountered during scanning
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return fileContent, nil
}

func write_file(args string) (string, error) {
	var writeFileArgs WriteFileArgs
	err := json.Unmarshal([]byte(args), &writeFileArgs)
	if err != nil {
		return "", err
	}

	if writeFileArgs.Path == "" {
		return "", errors.New("path is required")
	}

	file, err := os.Create(writeFileArgs.Path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	// Write contents, even if empty
	_, err = file.WriteString(writeFileArgs.Contents)

	if err != nil {
		return "", err
	}

	return "Write Success!", nil
}

func delete_file(args string) (string, error) {
	var delFileArgs DeleteFileArgs
	err := json.Unmarshal([]byte(args), &delFileArgs)
	if err != nil {
		return "", err
	}

	if delFileArgs.Path == "" {
		return "", errors.New("path is required")
	}

	err = os.Remove(delFileArgs.Path)
	if err != nil {
		return "", err
	}
	return "File successfully deleted!", nil
}

func exec_command(args string) (string, error) {
	var execArgs ExecCommandArgs
	err := json.Unmarshal([]byte(args), &execArgs)
	if err != nil {
		return "", err
	}

	if execArgs.Command == "" {
		return "", errors.New("command is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	// First argument is the command, following arguments are its parameters
	cmd := exec.CommandContext(ctx, execArgs.Command, execArgs.Args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return string(output), errors.New("command timed out")
		}
		return string(output), err
	}

	return string(output), nil

}
