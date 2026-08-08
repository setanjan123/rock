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
	fmt.Println("| Rock v1.01 |")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := os.Getenv("API_KEY")
	baseURL := os.Getenv("API_BASE_URL")
	model := os.Getenv("MODEL")
	var chatResponse ChatCompletionResponse
	var messages []Message
	var tools []ToolDefinition
	read_system_prompt(&messages)
	contextUsage := total_estimated_tokens(messages) // running estimate, kept in sync with every append
	tools = get_tools()
	scanner := bufio.NewScanner(os.Stdin)
	var input string
	var is_toolcall_continue = false
	for true {
		if !is_toolcall_continue {
			fmt.Println()
			fmt.Println(promptSeparator)
			fmt.Println("Current context: ", get_context_usage(contextUsage))
			fmt.Println("────────────────────────────────────────")
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
			userMsg := Message{Role: "user", Content: input}
			messages = append(messages, userMsg)
			track_context_usage(&contextUsage, userMsg)

			limit := get_context_limit()
			if needs_compaction(contextUsage, limit) {
				fmt.Println("Context limit approaching, summarizing conversation...")
				if err := compact_context(&messages, &apiKey, &baseURL, &model); err != nil {
					fmt.Println("Compaction failed, continuing without summary:", err)
				} else {
					contextUsage = total_estimated_tokens(messages) // summary rewrote history, recompute
					fmt.Println("Context compacted successfully.")
				}
			}
		}
		chatResponse, err = call_ai(&messages, &apiKey, &baseURL, &model, &tools)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			handle_response(chatResponse, &messages, &is_toolcall_continue, &contextUsage)
		}
	}
}

func call_ai(messages *[]Message, apiKey *string, baseURL *string, model *string, tools *[]ToolDefinition) (ChatCompletionResponse, error) {
	var toolsSlice []ToolDefinition
	if tools != nil {
		toolsSlice = *tools
	}
	request := ChatCompletionRequest{
		Model:      *model,
		Messages:   *messages,
		Tools:      toolsSlice,
		ToolChoice: "auto",
		Stream:     false,
	}
	postBody, err := json.Marshal(request)
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
