package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const conversationTitleMax = 80

var errConversationNotFound = errors.New("conversation not found")

type ConversationSummary struct {
	ID        string
	Title     string
	CreatedAt int64
	UpdatedAt int64
}

func open_db(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			id         TEXT PRIMARY KEY,
			title      TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			messages   TEXT NOT NULL
		);
	`)
	return err
}

func generate_conversation_id() string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	for i, b := range buf {
		buf[i] = chars[int(b)%len(chars)]
	}
	return string(buf)
}

func derive_title(messages []Message) string {
	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		title := strings.TrimSpace(msg.Content)
		if title == "" {
			continue
		}
		title = strings.Join(strings.Fields(title), " ")
		if len(title) > conversationTitleMax {
			title = title[:conversationTitleMax]
		}
		return title
	}
	return "untitled"
}

func save_conversation(db *sql.DB, id string, messages []Message) error {
	encoded, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	title := derive_title(messages)
	now := time.Now().Unix()

	_, err = db.Exec(`
		INSERT INTO conversations (id, title, created_at, updated_at, messages)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			updated_at = excluded.updated_at,
			messages = excluded.messages
	`, id, title, now, now, string(encoded))

	return err
}

func load_conversation(db *sql.DB, id string) ([]Message, error) {
	var encoded string
	err := db.QueryRow(`SELECT messages FROM conversations WHERE id = ?`, id).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errConversationNotFound
	}
	if err != nil {
		return nil, err
	}

	var messages []Message
	if err := json.Unmarshal([]byte(encoded), &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func list_conversations(db *sql.DB) ([]ConversationSummary, error) {
	rows, err := db.Query(`
		SELECT id, title, created_at, updated_at
		FROM conversations
		ORDER BY updated_at DESC, rowid DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ConversationSummary
	for rows.Next() {
		var summary ConversationSummary
		if err := rows.Scan(&summary.ID, &summary.Title, &summary.CreatedAt, &summary.UpdatedAt); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}
