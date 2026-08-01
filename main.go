package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

const promptSeparator = "────────────────────────────────────────"

func main() {
	fmt.Println("| Rock v0.01 |")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := os.Getenv("API_KEY")
	baseURL := os.Getenv("API_BASE_URL")
	model := os.Getenv("MODEL")
	var chatResponse ChatCompletionResponse
	var messages []map[string]any
	var tools []ToolDefinition
	read_system_prompt(&messages)
	tools = get_tools()
	scanner := bufio.NewScanner(os.Stdin)
	var input string
	var is_toolcall_continue = false
	for true {
		if !is_toolcall_continue {
			fmt.Println()
			fmt.Println(promptSeparator)
			fmt.Print("You › ")
			if scanner.Scan() {
				input = scanner.Text()
				switch input {
				case "/exit":
					return
				}

			}
			fmt.Println(promptSeparator)
			fmt.Println("Agent is thinking...")
			message := make(map[string]any)
			message["role"] = "user"
			message["content"] = input
			messages = append(messages, message)
		}
		chatResponse, err = call_ai(&messages, &apiKey, &baseURL, &model, &tools)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			handle_response(chatResponse, &messages, &is_toolcall_continue)
		}
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

func call_ai(messages *[]map[string]any, apiKey *string, baseURL *string, model *string, tools *[]ToolDefinition) (ChatCompletionResponse, error) {
	postBody, err := json.Marshal(map[string]any{
		"model":       *model,
		"messages":    *messages,
		"tools":       *tools,
		"tool_choice": "auto",
		"stream":      false,
	})
	var chatResponse ChatCompletionResponse
	if err != nil {
		return chatResponse, err
	}

	responseBody := bytes.NewBuffer(postBody)
	//Leverage Go's HTTP Post function to make request
	resp, err := http.NewRequest(http.MethodPost, *baseURL, responseBody)
	if err != nil {
		return chatResponse, err
	}
	resp.Header.Add("Content-Type", "application/json")
	resp.Header.Add("Authorization", "Bearer "+*apiKey)

	response, err := http.DefaultClient.Do(resp)

	//Handle Error
	if err != nil {
		return chatResponse, err
	}

	defer response.Body.Close()
	//Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return chatResponse, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return chatResponse, errors.New(string(body))
	}
	if err := json.Unmarshal(body, &chatResponse); err != nil { // Parse []byte to go struct pointer
		return chatResponse, err
	}
	if len(chatResponse.Choices) < 1 {
		return chatResponse, errors.New("Empty response")
	}
	return chatResponse, nil
}
