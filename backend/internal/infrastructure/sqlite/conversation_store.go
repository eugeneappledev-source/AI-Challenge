package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	_ "modernc.org/sqlite"
)

type ConversationStore struct{ db *sql.DB }

func Open(path string) (*ConversationStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open agent database: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &ConversationStore{db: db}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *ConversationStore) Close() error { return s.db.Close() }

func (s *ConversationStore) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		PRAGMA journal_mode = WAL;
		CREATE TABLE IF NOT EXISTS agent_messages (
			sequence INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			message_id TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			prompt_tokens INTEGER,
			completion_tokens INTEGER,
			total_tokens INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_agent_messages_conversation
		ON agent_messages(conversation_id, agent_id, sequence);
	`)
	if err != nil {
		return fmt.Errorf("migrate agent database: %w", err)
	}
	return nil
}

func (s *ConversationStore) Load(ctx context.Context, conversationID, agentID string) (domain.AgentConversation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT message_id, role, content, created_at, prompt_tokens, completion_tokens, total_tokens
		FROM agent_messages WHERE conversation_id = ? AND agent_id = ?
		ORDER BY sequence`, conversationID, agentID)
	if err != nil {
		return domain.AgentConversation{}, fmt.Errorf("load conversation: %w", err)
	}
	defer rows.Close()

	conversation := domain.AgentConversation{ID: conversationID, AgentID: agentID, Messages: []domain.AgentMessage{}}
	for rows.Next() {
		var message domain.AgentMessage
		var createdAt string
		var promptTokens, completionTokens, totalTokens sql.NullInt64
		if err := rows.Scan(&message.ID, &message.Role, &message.Content, &createdAt, &promptTokens, &completionTokens, &totalTokens); err != nil {
			return domain.AgentConversation{}, fmt.Errorf("scan conversation: %w", err)
		}
		message.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return domain.AgentConversation{}, fmt.Errorf("parse message time: %w", err)
		}
		if totalTokens.Valid {
			message.Usage = &domain.Usage{PromptTokens: int(promptTokens.Int64), CompletionTokens: int(completionTokens.Int64), TotalTokens: int(totalTokens.Int64)}
		}
		conversation.Messages = append(conversation.Messages, message)
		updatedAt := message.CreatedAt
		conversation.UpdatedAt = &updatedAt
	}
	if err := rows.Err(); err != nil {
		return domain.AgentConversation{}, fmt.Errorf("iterate conversation: %w", err)
	}
	return conversation, nil
}

func (s *ConversationStore) Append(ctx context.Context, conversationID, agentID string, messages ...domain.AgentMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin append messages: %w", err)
	}
	defer tx.Rollback()
	for _, message := range messages {
		var promptTokens, completionTokens, totalTokens any
		if message.Usage != nil {
			promptTokens, completionTokens, totalTokens = message.Usage.PromptTokens, message.Usage.CompletionTokens, message.Usage.TotalTokens
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO agent_messages
			(conversation_id, agent_id, message_id, role, content, created_at, prompt_tokens, completion_tokens, total_tokens)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			conversationID, agentID, message.ID, message.Role, message.Content, message.CreatedAt.Format(time.RFC3339Nano), promptTokens, completionTokens, totalTokens,
		); err != nil {
			return fmt.Errorf("append message: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit messages: %w", err)
	}
	return nil
}

func (s *ConversationStore) Clear(ctx context.Context, conversationID, agentID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM agent_messages WHERE conversation_id = ? AND agent_id = ?`, conversationID, agentID)
	if err != nil {
		return fmt.Errorf("clear conversation: %w", err)
	}
	return nil
}
