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

type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := os.Getenv("API_KEY")
	baseURL := os.Getenv("API_BASE_URL")
	var msg string
	var messages []map[string]string
	for true {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print(">")

		if scanner.Scan() {
			input := scanner.Text()
			if input == "/exit" {
				return
			}
			fmt.Println("Processing....")
			message := make(map[string]string)
			message["role"] = "user"
			message["content"] = input
			messages = append(messages, message)
			msg, err = callAI(&messages, &apiKey, &baseURL)
			if err != nil {
				msg = err.Error()
			} else {
				message := make(map[string]string)
				message["role"] = "assistant"
				message["content"] = msg
				messages = append(messages, message)
			}
		}
		fmt.Println(msg)
	}
}

func callAI(messages *[]map[string]string, apiKey *string, baseURL *string) (string, error) {
	postBody, err := json.Marshal(map[string]any{
		"model":    "openrouter/free",
		"messages": *messages,
		"stream":   false,
	})
	if err != nil {
		return "", err
	}

	responseBody := bytes.NewBuffer(postBody)
	//Leverage Go's HTTP Post function to make request
	resp, err := http.NewRequest(http.MethodPost, *baseURL, responseBody)
	if err != nil {
		return "", err
	}
	resp.Header.Add("Content-Type", "application/json")
	resp.Header.Add("Authorization", "Bearer "+*apiKey)

	response, err := http.DefaultClient.Do(resp)

	//Handle Error
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	//Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", errors.New(string(body))
	}
	var result ChatCompletionResponse
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		return "", err
	}
	if len(result.Choices) < 1 {
		return "", errors.New("Empty response")
	}
	return result.Choices[0].Message.Content, nil
}
