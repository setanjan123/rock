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
	msg := "Hello. Send your message."
	for true {
		fmt.Println(msg)
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print(">")

		if scanner.Scan() {
			input := scanner.Text()
			if input == "/exit" {
				return
			}
			fmt.Println("Processing....")
			msg, err = callAI(input, &apiKey, &baseURL)
			if err != nil {
				msg = err.Error()
			}
		}
	}
}

func callAI(message string, apiKey *string, baseURL *string) (string, error) {
	postBody, err := json.Marshal(map[string]any{
		"model": "openrouter/free",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": message,
			},
		},
		"stream": false,
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
