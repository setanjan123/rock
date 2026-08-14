package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatCompletionsURL(t *testing.T) {
	got := chat_completions_url("http://localhost:11434")
	want := "http://localhost:11434/v1/chat/completions"
	if got != want {
		t.Fatalf("chat_completions_url = %q, want %q", got, want)
	}

	if got := chat_completions_url("http://localhost:11434/"); got != want {
		t.Fatalf("chat_completions_url with trailing slash = %q, want %q", got, want)
	}
}

func TestModelsURL(t *testing.T) {
	got := models_url("http://localhost:11434")
	want := "http://localhost:11434/v1/models"
	if got != want {
		t.Fatalf("models_url = %q, want %q", got, want)
	}

	if got := models_url("http://localhost:11434/"); got != want {
		t.Fatalf("models_url with trailing slash = %q, want %q", got, want)
	}
}

func TestResolveModel(t *testing.T) {
	models := []string{"alpha", "beta"}

	cases := []struct {
		name    string
		choice  string
		want    string
		wantErr bool
	}{
		{name: "number", choice: "2", want: "beta"},
		{name: "exact id", choice: "alpha", want: "alpha"},
		{name: "trims whitespace", choice: "  beta  ", want: "beta"},
		{name: "out of range", choice: "3", wantErr: true},
		{name: "unknown id", choice: "nope", wantErr: true},
		{name: "empty", choice: "", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolve_model(models, tc.choice)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolve_model(%q) expected error", tc.choice)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve_model(%q) unexpected error: %v", tc.choice, err)
			}
			if got != tc.want {
				t.Fatalf("resolve_model(%q) = %q, want %q", tc.choice, got, tc.want)
			}
		})
	}
}

func TestFetchModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4o","object":"model","created":1,"owned_by":"openai"}]}`))
	}))
	defer server.Close()

	models, err := fetch_models("key", server.URL)
	if err != nil {
		t.Fatalf("fetch_models failed: %v", err)
	}
	if len(models) != 1 || models[0] != "gpt-4o" {
		t.Fatalf("unexpected models: %+v", models)
	}
}

func TestFetchModelsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	_, err := fetch_models("bad", server.URL)
	if err == nil {
		t.Fatal("fetch_models expected error")
	}
}
