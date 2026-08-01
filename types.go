package main

type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
	Model   string   `json:"model"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
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
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ListDirectoryArgs struct {
	Path string `json:"path"`
}

type ListDirectoryResult struct {
	Path    string   `json:"path"`
	Entries []string `json:"entries"`
}
