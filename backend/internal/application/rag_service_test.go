package application

import (
	"context"
	"strings"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type ragStoreStub struct{ chunks []domain.KnowledgeChunk }

func (s *ragStoreStub) LoadKnowledgeChunks(context.Context, domain.ChunkStrategy) ([]domain.KnowledgeChunk, error) {
	return s.chunks, nil
}

type ragIndexerStub struct{}

func (ragIndexerStub) Build(context.Context) (domain.KnowledgeIndexStatus, error) {
	return domain.KnowledgeIndexStatus{}, nil
}

type ragModelStub struct{ prompts []string }

func (s *ragModelStub) Generate(_ context.Context, request domain.ModelRequest) (domain.ModelResponse, error) {
	s.prompts = append(s.prompts, request.UserPrompt)
	return domain.ModelResponse{Content: "test answer", Model: "test-model", Usage: domain.Usage{TotalTokens: 12}}, nil
}

func TestRAGCompareUsesRetrievedRepositoryContext(t *testing.T) {
	content := "Caddy получает HTTPS и проксирует запросы к Go API."
	store := &ragStoreStub{chunks: []domain.KnowledgeChunk{{ID: "one", Strategy: domain.ChunkStrategyStructural, Source: "deploy/Caddyfile", Section: "gateway", Content: content, Vector: embedText(content), Dimensions: 256}}}
	model := &ragModelStub{}
	service := NewRAGService(store, ragIndexerStub{}, model, 500)
	result, err := service.Compare(context.Background(), "Как проект получает HTTPS?")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Retrieved) != 1 || result.Retrieved[0].Chunk.Source != "deploy/Caddyfile" {
		t.Fatalf("unexpected retrieval: %#v", result.Retrieved)
	}
	if len(model.prompts) != 2 || !strings.Contains(model.prompts[1], content) {
		t.Fatalf("RAG context was not sent: %#v", model.prompts)
	}
	if result.WithoutRAG.Answer == "" || result.WithRAG.Answer == "" {
		t.Fatal("both comparison answers are required")
	}
}

func TestRAGControlSetHasTenQuestions(t *testing.T) {
	service := NewRAGService(&ragStoreStub{}, ragIndexerStub{}, &ragModelStub{}, 500)
	questions := service.ControlQuestions()
	if len(questions) != 10 {
		t.Fatalf("got %d questions", len(questions))
	}
	for _, question := range questions {
		if question.Question == "" || len(question.ExpectedSources) == 0 {
			t.Fatalf("invalid control question: %#v", question)
		}
	}
}

func TestRetrievalComparisonRewritesReranksAndFilters(t *testing.T) {
	chunks := []domain.KnowledgeChunk{}
	for _, item := range []struct{ id, source, content string }{
		{"https", "deploy/Caddyfile", "Caddy reverse proxy automatically provisions HTTPS certificates."},
		{"other", "README.md", "The project contains an iOS application."},
	} {
		chunks = append(chunks, domain.KnowledgeChunk{ID: item.id, Strategy: domain.ChunkStrategyStructural, Source: item.source, Content: item.content, Vector: embedText(item.content), Dimensions: 256})
	}
	model := &ragModelStub{}
	service := NewRAGService(&ragStoreStub{chunks: chunks}, ragIndexerStub{}, model, 500)
	result, err := service.CompareRetrieval(context.Background(), "Как работает HTTPS?", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.RewrittenQuery == "" || len(result.Candidates) != 2 || len(result.Improved) < 1 {
		t.Fatalf("unexpected comparison: %#v", result)
	}
	if result.Candidates[0].RerankScore < result.Candidates[1].RerankScore {
		t.Fatal("candidates are not reranked")
	}
}
