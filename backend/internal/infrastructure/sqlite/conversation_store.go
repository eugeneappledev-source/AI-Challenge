package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
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
		CREATE TABLE IF NOT EXISTS agent_facts (
			session_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			facts_json TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (session_id, agent_id)
		);
		CREATE TABLE IF NOT EXISTS agent_working_memory (
			task_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			memory_key TEXT NOT NULL,
			memory_value TEXT NOT NULL,
			source TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (task_id, agent_id, memory_key)
		);
		CREATE TABLE IF NOT EXISTS agent_long_term_memory (
			user_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			memory_key TEXT NOT NULL,
			memory_value TEXT NOT NULL,
			source TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (user_id, agent_id, memory_key)
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate agent database: %w", err)
	}
	return nil
}

func (s *ConversationStore) LoadWorkingMemory(ctx context.Context, taskID, agentID string) ([]domain.MemoryItem, error) {
	return s.loadMemoryItems(ctx, "agent_working_memory", "task_id", taskID, agentID)
}

func (s *ConversationStore) SaveWorkingMemory(ctx context.Context, taskID, agentID string, item domain.MemoryItem) error {
	return s.saveMemoryItem(ctx, "agent_working_memory", "task_id", taskID, agentID, item)
}

func (s *ConversationStore) ClearWorkingMemory(ctx context.Context, taskID, agentID string) error {
	return s.clearMemoryItems(ctx, "agent_working_memory", "task_id", taskID, agentID)
}

func (s *ConversationStore) LoadLongTermMemory(ctx context.Context, userID, agentID string) ([]domain.MemoryItem, error) {
	return s.loadMemoryItems(ctx, "agent_long_term_memory", "user_id", userID, agentID)
}

func (s *ConversationStore) SaveLongTermMemory(ctx context.Context, userID, agentID string, item domain.MemoryItem) error {
	return s.saveMemoryItem(ctx, "agent_long_term_memory", "user_id", userID, agentID, item)
}

func (s *ConversationStore) ClearLongTermMemory(ctx context.Context, userID, agentID string) error {
	return s.clearMemoryItems(ctx, "agent_long_term_memory", "user_id", userID, agentID)
}

func (s *ConversationStore) loadMemoryItems(ctx context.Context, table, scopeColumn, scopeID, agentID string) ([]domain.MemoryItem, error) {
	query := fmt.Sprintf(`SELECT memory_key, memory_value, source, updated_at FROM %s WHERE %s = ? AND agent_id = ? ORDER BY updated_at DESC`, table, scopeColumn)
	rows, err := s.db.QueryContext(ctx, query, scopeID, agentID)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", table, err)
	}
	defer rows.Close()
	items := []domain.MemoryItem{}
	for rows.Next() {
		var item domain.MemoryItem
		var updatedAt string
		if err := rows.Scan(&item.Key, &item.Value, &item.Source, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan %s: %w", table, err)
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse %s time: %w", table, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s: %w", table, err)
	}
	return items, nil
}

func (s *ConversationStore) saveMemoryItem(ctx context.Context, table, scopeColumn, scopeID, agentID string, item domain.MemoryItem) error {
	query := fmt.Sprintf(`INSERT INTO %s (%s, agent_id, memory_key, memory_value, source, updated_at) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(%s, agent_id, memory_key) DO UPDATE SET memory_value = excluded.memory_value, source = excluded.source, updated_at = excluded.updated_at`, table, scopeColumn, scopeColumn)
	if _, err := s.db.ExecContext(ctx, query, scopeID, agentID, item.Key, item.Value, item.Source, item.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("save %s: %w", table, err)
	}
	return nil
}

func (s *ConversationStore) clearMemoryItems(ctx context.Context, table, scopeColumn, scopeID, agentID string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = ? AND agent_id = ?`, table, scopeColumn)
	if _, err := s.db.ExecContext(ctx, query, scopeID, agentID); err != nil {
		return fmt.Errorf("clear %s: %w", table, err)
	}
	return nil
}

func (s *ConversationStore) Trim(ctx context.Context, conversationID, agentID string, keep int) error {
	if keep < 0 {
		keep = 0
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM agent_messages
		WHERE conversation_id = ? AND agent_id = ? AND sequence NOT IN (
			SELECT sequence FROM agent_messages WHERE conversation_id = ? AND agent_id = ?
			ORDER BY sequence DESC LIMIT ?
		)`, conversationID, agentID, conversationID, agentID, keep)
	if err != nil {
		return fmt.Errorf("trim conversation: %w", err)
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

func (s *ConversationStore) LoadFacts(ctx context.Context, sessionID, agentID string) (map[string]string, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT facts_json FROM agent_facts WHERE session_id = ? AND agent_id = ?`, sessionID, agentID).Scan(&raw)
	if err == sql.ErrNoRows {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load agent facts: %w", err)
	}
	facts := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &facts); err != nil {
		return nil, fmt.Errorf("decode agent facts: %w", err)
	}
	return facts, nil
}

func (s *ConversationStore) SaveFacts(ctx context.Context, sessionID, agentID string, facts map[string]string, updatedAt time.Time) error {
	raw, err := json.Marshal(facts)
	if err != nil {
		return fmt.Errorf("encode agent facts: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO agent_facts
		(session_id, agent_id, facts_json, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(session_id, agent_id) DO UPDATE SET
		facts_json = excluded.facts_json, updated_at = excluded.updated_at`,
		sessionID, agentID, string(raw), updatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save agent facts: %w", err)
	}
	return nil
}

func (s *ConversationStore) ClearFacts(ctx context.Context, sessionID, agentID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM agent_facts WHERE session_id = ? AND agent_id = ?`, sessionID, agentID); err != nil {
		return fmt.Errorf("clear agent facts: %w", err)
	}
	return nil
}
