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
			total_tokens INTEGER,
			cache_hit_tokens INTEGER,
			cache_miss_tokens INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_agent_messages_conversation
		ON agent_messages(conversation_id, agent_id, sequence);
		CREATE TABLE IF NOT EXISTS agent_summaries (
			conversation_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			content TEXT NOT NULL,
			covered_messages INTEGER NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (conversation_id, agent_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate agent database: %w", err)
	}
	return nil
}

func (s *ConversationStore) Load(ctx context.Context, conversationID, agentID string) (domain.AgentConversation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT message_id, role, content, created_at, prompt_tokens, completion_tokens, total_tokens, cache_hit_tokens, cache_miss_tokens
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
		var promptTokens, completionTokens, totalTokens, cacheHitTokens, cacheMissTokens sql.NullInt64
		if err := rows.Scan(&message.ID, &message.Role, &message.Content, &createdAt, &promptTokens, &completionTokens, &totalTokens, &cacheHitTokens, &cacheMissTokens); err != nil {
			return domain.AgentConversation{}, fmt.Errorf("scan conversation: %w", err)
		}
		message.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return domain.AgentConversation{}, fmt.Errorf("parse message time: %w", err)
		}
		if totalTokens.Valid {
			message.Usage = &domain.Usage{
				PromptTokens: int(promptTokens.Int64), CompletionTokens: int(completionTokens.Int64), TotalTokens: int(totalTokens.Int64),
				PromptCacheHitTokens: int(cacheHitTokens.Int64), PromptCacheMissTokens: int(cacheMissTokens.Int64),
			}
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
		var promptTokens, completionTokens, totalTokens, cacheHitTokens, cacheMissTokens any
		if message.Usage != nil {
			promptTokens, completionTokens, totalTokens = message.Usage.PromptTokens, message.Usage.CompletionTokens, message.Usage.TotalTokens
			cacheHitTokens, cacheMissTokens = message.Usage.PromptCacheHitTokens, message.Usage.PromptCacheMissTokens
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO agent_messages
			(conversation_id, agent_id, message_id, role, content, created_at, prompt_tokens, completion_tokens, total_tokens, cache_hit_tokens, cache_miss_tokens)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			conversationID, agentID, message.ID, message.Role, message.Content, message.CreatedAt.Format(time.RFC3339Nano), promptTokens, completionTokens, totalTokens, cacheHitTokens, cacheMissTokens,
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin clear conversation: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_messages WHERE conversation_id = ? AND agent_id = ?`, conversationID, agentID); err != nil {
		return fmt.Errorf("clear conversation messages: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_summaries WHERE conversation_id = ? AND agent_id = ?`, conversationID, agentID); err != nil {
		return fmt.Errorf("clear conversation summary: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit clear conversation: %w", err)
	}
	return nil
}

func (s *ConversationStore) LoadSummary(ctx context.Context, conversationID, agentID string) (domain.ConversationSummary, error) {
	var summary domain.ConversationSummary
	var updatedAt string
	err := s.db.QueryRowContext(ctx, `SELECT content, covered_messages, updated_at FROM agent_summaries WHERE conversation_id = ? AND agent_id = ?`, conversationID, agentID).
		Scan(&summary.Content, &summary.CoveredMessages, &updatedAt)
	if err == sql.ErrNoRows {
		return domain.ConversationSummary{ConversationID: conversationID, AgentID: agentID}, nil
	}
	if err != nil {
		return domain.ConversationSummary{}, fmt.Errorf("load summary: %w", err)
	}
	summary.ConversationID, summary.AgentID = conversationID, agentID
	summary.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return domain.ConversationSummary{}, fmt.Errorf("parse summary time: %w", err)
	}
	return summary, nil
}

func (s *ConversationStore) SaveSummary(ctx context.Context, summary domain.ConversationSummary) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_summaries
		(conversation_id, agent_id, content, covered_messages, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(conversation_id, agent_id) DO UPDATE SET
		content = excluded.content, covered_messages = excluded.covered_messages, updated_at = excluded.updated_at`,
		summary.ConversationID, summary.AgentID, summary.Content, summary.CoveredMessages, summary.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save summary: %w", err)
	}
	return nil
}
