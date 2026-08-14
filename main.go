package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const promptSeparator = "────────────────────────────────────────"

func main() {
	fmt.Println("| Rock v1.01 |")
	apiKey, baseURL, model, err := load_config()
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := open_db("rock.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	var conversationID string
	var messages []Message

	if len(os.Args) > 1 {
		conversationID = os.Args[1]
		messages, storedModel, err := load_conversation(db, conversationID)
		if err != nil {
			log.Fatalf("failed to resume conversation %q: %v", conversationID, err)
		}
		if storedModel != "" {
			model = storedModel
		}
		fmt.Println("Resumed conversation", conversationID)
		print_conversation(messages)
	} else {
		conversationID = generate_conversation_id()
		read_system_prompt(&messages)
		fmt.Println("New conversation", conversationID)
	}

	var tools []ToolDefinition
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
			fmt.Println("Conversation: ", conversationID)
			fmt.Println("Model: ", model)
			fmt.Println(promptSeparator)
			fmt.Print("You › ")
			if scanner.Scan() {
				input = scanner.Text()
				switch {
				case input == "/exit":
					if err := save_conversation(db, conversationID, model, messages); err != nil {
						fmt.Println("Failed to save conversation:", err)
					} else {
						fmt.Println("Saved as", conversationID, "- resume with: rock", conversationID)
					}
					return
				case input == "/history":
					print_history(db, conversationID)
					continue
				case input == "/model":
					if err := select_model(apiKey, baseURL, &model, scanner); err != nil {
						fmt.Println("Failed to switch model:", err)
					} else if err := save_conversation(db, conversationID, model, messages); err != nil {
						fmt.Println("Failed to persist model choice:", err)
					}
					continue
				case strings.HasPrefix(input, "/model "):
					choice := strings.TrimSpace(strings.TrimPrefix(input, "/model "))
					if err := switch_model(apiKey, baseURL, &model, choice); err != nil {
						fmt.Println("Failed to switch model:", err)
					} else if err := save_conversation(db, conversationID, model, messages); err != nil {
						fmt.Println("Failed to persist model choice:", err)
					}
					continue
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
		chatResponse, err := call_ai(&messages, &apiKey, &baseURL, &model, &tools)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			handle_response(chatResponse, &messages, &is_toolcall_continue, &contextUsage)
		}

		if !is_toolcall_continue {
			if err := save_conversation(db, conversationID, model, messages); err != nil {
				fmt.Println("Failed to save conversation:", err)
			}
		}
	}
}

func print_conversation(messages []Message) {
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			continue
		case "user":
			fmt.Println(promptSeparator)
			fmt.Println("You ›")
			fmt.Println(msg.Content)
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				print_tool_calls(msg.ToolCalls)
			} else {
				fmt.Println(promptSeparator)
				fmt.Println("Agent ›")
				fmt.Println(msg.Content)
			}
		case "tool":
			if verbose_tools() {
				fmt.Println(promptSeparator)
				fmt.Println("Tool result ›")
				fmt.Println(msg.Content)
			}
		}
	}
}

func print_history(db *sql.DB, currentID string) {
	summaries, err := list_conversations(db)
	if err != nil {
		fmt.Println("Failed to list conversations:", err)
		return
	}
	if len(summaries) == 0 {
		fmt.Println("No past conversations yet.")
		return
	}
	fmt.Println()
	fmt.Println("Past conversations:")
	for _, summary := range summaries {
		marker := " "
		if summary.ID == currentID {
			marker = "*"
		}
		updated := time.Unix(summary.UpdatedAt, 0).Format("2006-01-02 15:04")
		fmt.Printf("%s %s  %s  %s\n", marker, summary.ID, summary.Title, updated)
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
	resp, err := http.NewRequest(http.MethodPost, chat_completions_url(*baseURL), responseBody)
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
