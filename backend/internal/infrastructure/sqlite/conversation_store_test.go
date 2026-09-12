package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func TestConversationSurvivesStoreReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	usage := domain.Usage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 14, PromptCacheHitTokens: 6, PromptCacheMissTokens: 4}
	err = store.Append(context.Background(), "c1", "mentor",
		domain.AgentMessage{ID: "u1", Role: "user", Content: "Меня зовут Женя", CreatedAt: now},
		domain.AgentMessage{ID: "a1", Role: "assistant", Content: "Запомнил", CreatedAt: now.Add(time.Nanosecond), Usage: &usage},
	)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	conversation, err := reopened.Load(context.Background(), "c1", "mentor")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(conversation.Messages) != 2 || conversation.Messages[0].Content != "Меня зовут Женя" || conversation.Messages[1].Usage.TotalTokens != 14 || conversation.Messages[1].Usage.PromptCacheHitTokens != 6 {
		t.Fatalf("unexpected restored conversation: %+v", conversation)
	}

	if err := reopened.Clear(context.Background(), "c1", "mentor"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	conversation, err = reopened.Load(context.Background(), "c1", "mentor")
	if err != nil || len(conversation.Messages) != 0 {
		t.Fatalf("expected empty conversation, got %+v, %v", conversation, err)
	}
}

func TestSummaryIsStoredSeparatelyAndUpdated(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()
	summary := domain.ConversationSummary{
		ConversationID: "c1", AgentID: "mentor", Content: "Пользователь любит Swift",
		CoveredMessages: 10, UpdatedAt: time.Now().UTC(),
	}
	if err := store.SaveSummary(context.Background(), summary); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := store.LoadSummary(context.Background(), "c1", "mentor")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Content != summary.Content || loaded.CoveredMessages != 10 {
		t.Fatalf("unexpected summary: %+v", loaded)
	}
}
