package main

type ChatCompletionRequest struct {
	Model      string           `json:"model"`
	Messages   []Message        `json:"messages"`
	Tools      []ToolDefinition `json:"tools"`
	ToolChoice string           `json:"tool_choice"`
	Stream     bool             `json:"stream"`
}

type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
	Model   string   `json:"model"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
}

type ToolParameters struct {
	Type                 string                  `json:"type"`
	Properties           map[string]ToolProperty `json:"properties"`
	Required             []string                `json:"required"`
	AdditionalProperties bool                    `json:"additionalProperties"`
}

type ToolProperty struct {
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Items       *ToolProperty `json:"items,omitempty"`
}

type ListDirectoryArgs struct {
	Path string `json:"path"`
}

type ReadFileArgs struct {
	Path string `json:"path"`
}

type WriteFileArgs struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

type DeleteFileArgs struct {
	Path string `json:"path"`
}

type ExecCommandArgs struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type ListDirectoryResult struct {
	Path    string   `json:"path"`
	Entries []string `json:"entries"`
}

type ToolCallResponse struct {
	Index    int
	Response string
	Error    error
}

type ModelsListResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID string `json:"id"`
	} `json:"data"`
}

