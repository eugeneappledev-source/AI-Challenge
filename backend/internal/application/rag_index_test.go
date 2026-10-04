package application_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type memoryKnowledgeStore struct {
	chunks map[domain.ChunkStrategy][]domain.KnowledgeChunk
}

func (s *memoryKnowledgeStore) ReplaceKnowledgeChunks(_ context.Context, strategy domain.ChunkStrategy, chunks []domain.KnowledgeChunk) error {
	if s.chunks == nil {
		s.chunks = map[domain.ChunkStrategy][]domain.KnowledgeChunk{}
	}
	s.chunks[strategy] = chunks
	return nil
}
func (s *memoryKnowledgeStore) KnowledgeIndexStatus(_ context.Context, path string) (domain.KnowledgeIndexStatus, error) {
	status := domain.KnowledgeIndexStatus{CorpusPath: path}
	for strategy, chunks := range s.chunks {
		status.Strategies = append(status.Strategies, domain.IndexStrategyStats{Strategy: strategy, Chunks: len(chunks), Dimensions: 256})
	}
	return status, nil
}

func TestKnowledgeIndexerBuildsTwoStrategiesWithEmbeddings(t *testing.T) {
	root := t.TempDir()
	content := "# Architecture\n\nThe API uses Go and SQLite.\n\n## Memory\n\nConversation data survives restarts.\n"
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &memoryKnowledgeStore{}
	service := application.NewKnowledgeIndexService(store, root)
	if _, err := service.Build(context.Background()); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	for _, strategy := range []domain.ChunkStrategy{domain.ChunkStrategyFixed, domain.ChunkStrategyStructural} {
		chunks := store.chunks[strategy]
		if len(chunks) == 0 {
			t.Fatalf("no chunks for %s", strategy)
		}
		if len(chunks[0].Vector) != 256 {
			t.Fatalf("unexpected embedding dimensions: %d", len(chunks[0].Vector))
		}
		if chunks[0].Source != "README.md" || chunks[0].ID == "" {
			t.Fatalf("metadata missing: %+v", chunks[0])
		}
	}
}
