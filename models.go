package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// chat_completions_url builds the chat-completions endpoint from the base URL.
func chat_completions_url(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
}

// models_url builds the List Models endpoint from the base URL.
func models_url(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/v1/models"
}

// fetch_models requests the available model ids from the OpenAI-compatible
// List Models endpoint.
func fetch_models(apiKey, baseURL string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, models_url(baseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(string(body))
	}

	var list ModelsListResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, err
	}
	if len(list.Data) == 0 {
		return nil, errors.New("no models returned")
	}

	ids := make([]string, 0, len(list.Data))
	for _, model := range list.Data {
		ids = append(ids, model.ID)
	}
	return ids, nil
}

// resolve_model maps a user choice to a model id. A numeric choice selects a
// 1-based entry from the list; any other value must match a model id exactly.
func resolve_model(models []string, choice string) (string, error) {
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return "", errors.New("no model selected")
	}

	if index, err := strconv.Atoi(choice); err == nil {
		if index < 1 || index > len(models) {
			return "", fmt.Errorf("selection out of range (1-%d)", len(models))
		}
		return models[index-1], nil
	}

	for _, id := range models {
		if id == choice {
			return choice, nil
		}
	}
	return "", fmt.Errorf("model %q not found", choice)
}

// print_models displays the available models as a numbered list.
func print_models(models []string) {
	fmt.Println("Available models:")
	for i, id := range models {
		fmt.Printf("  %d. %s\n", i+1, id)
	}
}

// set_model resolves the choice against already-fetched models and updates the
// active model.
func set_model(models []string, model *string, choice string) error {
	resolved, err := resolve_model(models, choice)
	if err != nil {
		return err
	}

	*model = resolved
	fmt.Println("Model set to", resolved)
	return nil
}

// select_model lists available models, prompts for a choice, and applies it.
func select_model(apiKey, baseURL string, model *string, scanner *bufio.Scanner) error {
	models, err := fetch_models(apiKey, baseURL)
	if err != nil {
		return err
	}

	print_models(models)
	fmt.Print("Select a model (number or id): ")
	if !scanner.Scan() {
		return errors.New("no input")
	}

	return set_model(models, model, scanner.Text())
}

// switch_model switches directly to the requested model id without prompting.
func switch_model(apiKey, baseURL string, model *string, choice string) error {
	models, err := fetch_models(apiKey, baseURL)
	if err != nil {
		return err
	}

	return set_model(models, model, choice)
}
