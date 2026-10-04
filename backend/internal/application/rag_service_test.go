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
