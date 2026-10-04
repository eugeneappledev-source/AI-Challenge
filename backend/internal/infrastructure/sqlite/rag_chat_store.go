package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func (s *ConversationStore) AppendRAGChatMessage(ctx context.Context, sessionID string, message domain.RAGChatMessage) (domain.RAGChatMessage, error) {
	citations, err := json.Marshal(message.Citations)
	if err != nil {
		return message, fmt.Errorf("encode RAG citations: %w", err)
	}
	usage, err := json.Marshal(message.Usage)
	if err != nil {
		return message, fmt.Errorf("encode RAG usage: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO rag_chat_messages(session_id,role,content,citations_json,usage_json,created_at) VALUES(?,?,?,?,?,?)`, sessionID, message.Role, message.Content, string(citations), string(usage), message.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return message, fmt.Errorf("append RAG chat message: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return message, err
	}
	message.Sequence = int(id)
	return message, nil
}

func (s *ConversationStore) LoadRAGChat(ctx context.Context, sessionID string) ([]domain.RAGChatMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT sequence,role,content,citations_json,usage_json,created_at FROM rag_chat_messages WHERE session_id=? ORDER BY sequence`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load RAG chat: %w", err)
	}
	defer rows.Close()
	messages := []domain.RAGChatMessage{}
	for rows.Next() {
		var message domain.RAGChatMessage
		var citationsRaw, usageRaw, created string
		if err := rows.Scan(&message.Sequence, &message.Role, &message.Content, &citationsRaw, &usageRaw, &created); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(citationsRaw), &message.Citations); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(usageRaw), &message.Usage); err != nil {
			return nil, err
		}
		message.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *ConversationStore) LoadRAGTaskState(ctx context.Context, sessionID string) (domain.RAGTaskState, bool, error) {
	var state domain.RAGTaskState
	var constraintsRaw, termsRaw, updated string
	err := s.db.QueryRowContext(ctx, `SELECT goal,constraints_json,terms_json,updated_at FROM rag_task_states WHERE session_id=?`, sessionID).Scan(&state.Goal, &constraintsRaw, &termsRaw, &updated)
	if err == sql.ErrNoRows {
		return domain.RAGTaskState{SessionID: sessionID, Constraints: []string{}, Terms: map[string]string{}}, false, nil
	}
	if err != nil {
		return state, false, fmt.Errorf("load RAG task state: %w", err)
	}
	state.SessionID = sessionID
	if err := json.Unmarshal([]byte(constraintsRaw), &state.Constraints); err != nil {
		return state, false, err
	}
	if err := json.Unmarshal([]byte(termsRaw), &state.Terms); err != nil {
		return state, false, err
	}
	state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return state, true, nil
}

func (s *ConversationStore) SaveRAGTaskState(ctx context.Context, state domain.RAGTaskState) error {
	constraints, err := json.Marshal(state.Constraints)
	if err != nil {
		return err
	}
	terms, err := json.Marshal(state.Terms)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO rag_task_states(session_id,goal,constraints_json,terms_json,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(session_id) DO UPDATE SET goal=excluded.goal,constraints_json=excluded.constraints_json,terms_json=excluded.terms_json,updated_at=excluded.updated_at`, state.SessionID, state.Goal, string(constraints), string(terms), state.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save RAG task state: %w", err)
	}
	return nil
}

func (s *ConversationStore) ClearRAGChat(ctx context.Context, sessionID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM rag_chat_messages WHERE session_id=?`, sessionID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rag_task_states WHERE session_id=?`, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}
