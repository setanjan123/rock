package main

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// test_db opens a temporary SQLite database for a test and registers cleanup.
func test_db(t *testing.T) *sql.DB {
	t.Helper()
	db, err := open_db(filepath.Join(t.TempDir(), "rock.db"))
	if err != nil {
		t.Fatalf("open_db failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSaveAndLoadConversation(t *testing.T) {
	db := test_db(t)

	messages := []Message{
		{Role: "system", Content: "You are a helpful coding assistant."},
		{Role: "user", Content: "read foo.go"},
		{
			Role:    "assistant",
			Content: "",
			ToolCalls: []ToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: Function{
					Name:      "read_file",
					Arguments: `{"path":"foo.go"}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_1", Content: "package main"},
	}

	if err := save_conversation(db, "abc123", messages); err != nil {
		t.Fatalf("save_conversation failed: %v", err)
	}

	got, err := load_conversation(db, "abc123")
	if err != nil {
		t.Fatalf("load_conversation failed: %v", err)
	}

	if len(got) != len(messages) {
		t.Fatalf("loaded %d messages, want %d", len(got), len(messages))
	}
	for i := range messages {
		if got[i].Role != messages[i].Role {
			t.Fatalf("message %d role = %q, want %q", i, got[i].Role, messages[i].Role)
		}
		if got[i].Content != messages[i].Content {
			t.Fatalf("message %d content = %q, want %q", i, got[i].Content, messages[i].Content)
		}
	}
	if len(got[2].ToolCalls) != 1 || got[2].ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("tool call not round-tripped: %+v", got[2].ToolCalls)
	}
}

func TestLoadMissingConversation(t *testing.T) {
	db := test_db(t)

	_, err := load_conversation(db, "nope")
	if !errors.Is(err, errConversationNotFound) {
		t.Fatalf("load_conversation error = %v, want errConversationNotFound", err)
	}
}

func TestSaveConversationUpdatesExisting(t *testing.T) {
	db := test_db(t)

	first := []Message{{Role: "user", Content: "hello"}}
	if err := save_conversation(db, "id1", first); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	second := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	if err := save_conversation(db, "id1", second); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	got, err := load_conversation(db, "id1")
	if err != nil {
		t.Fatalf("load after update failed: %v", err)
	}
	if len(got) != len(second) {
		t.Fatalf("loaded %d messages after update, want %d", len(got), len(second))
	}

	summaries, err := list_conversations(db)
	if err != nil {
		t.Fatalf("list_conversations failed: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("list_conversations returned %d rows, want 1", len(summaries))
	}
}

func TestListConversationsOrdering(t *testing.T) {
	db := test_db(t)

	for _, id := range []string{"a", "b", "c"} {
		if err := save_conversation(db, id, []Message{{Role: "user", Content: id}}); err != nil {
			t.Fatalf("save %s failed: %v", id, err)
		}
	}

	summaries, err := list_conversations(db)
	if err != nil {
		t.Fatalf("list_conversations failed: %v", err)
	}
	if len(summaries) != 3 {
		t.Fatalf("list_conversations returned %d rows, want 3", len(summaries))
	}
	if summaries[0].ID != "c" || summaries[1].ID != "b" || summaries[2].ID != "a" {
		t.Fatalf("unexpected ordering: %+v", summaries)
	}
}

func TestDeriveTitle(t *testing.T) {
	cases := []struct {
		name     string
		messages []Message
		want     string
	}{
		{
			name:     "first user message",
			messages: []Message{{Role: "system", Content: "prompt"}, {Role: "user", Content: "  read file  "}},
			want:     "read file",
		},
		{
			name:     "skips empty user message",
			messages: []Message{{Role: "user", Content: "   "}, {Role: "user", Content: "next"}},
			want:     "next",
		},
		{
			name:     "truncates long title",
			messages: []Message{{Role: "user", Content: strings.Repeat("x", conversationTitleMax+50)}},
			want:     strings.Repeat("x", conversationTitleMax),
		},
		{
			name: "no user message falls back",
			want: "untitled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := derive_title(tc.messages); got != tc.want {
				t.Fatalf("derive_title = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGenerateConversationID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := generate_conversation_id()
		if id == "" {
			t.Fatalf("generate_conversation_id returned empty string")
		}
		if seen[id] {
			t.Fatalf("generate_conversation_id returned duplicate %q", id)
		}
		seen[id] = true
	}
}
