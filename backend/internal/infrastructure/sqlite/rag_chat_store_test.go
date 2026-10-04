package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func TestRAGChatPersistsAcrossStoreReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rag.db")
	ctx := context.Background()
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	state := domain.RAGTaskState{SessionID: "persist-session", Goal: "Ship RAG", Constraints: []string{"no new services"}, Terms: map[string]string{"gate": "refuse below threshold"}, UpdatedAt: time.Now().UTC()}
	if err := store.SaveRAGTaskState(ctx, state); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendRAGChatMessage(ctx, state.SessionID, domain.RAGChatMessage{Role: "user", Content: "Remember this", Citations: []domain.EvidenceCitation{}, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	messages, err := reopened.LoadRAGChat(ctx, state.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := reopened.LoadRAGTaskState(ctx, state.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || !ok || loaded.Goal != state.Goal || loaded.Terms["gate"] == "" {
		t.Fatalf("persistence failed: messages=%#v state=%#v", messages, loaded)
	}
}
