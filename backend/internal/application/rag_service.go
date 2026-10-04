package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var (
	ErrEmptyRAGQuestion   = errors.New("question is required")
	ErrRAGQuestionTooLong = errors.New("question is too long")
)

type RAGStore interface {
	LoadKnowledgeChunks(context.Context, domain.ChunkStrategy) ([]domain.KnowledgeChunk, error)
}

type RAGIndexer interface {
	Build(context.Context) (domain.KnowledgeIndexStatus, error)
}

type RAGService struct {
	store    RAGStore
	indexer  RAGIndexer
	client   ReasoningClient
	maxRunes int
}

func NewRAGService(store RAGStore, indexer RAGIndexer, client ReasoningClient, maxRunes int) *RAGService {
	return &RAGService{store: store, indexer: indexer, client: client, maxRunes: maxRunes}
}

func (s *RAGService) ControlQuestions() []domain.ControlQuestion {
	return []domain.ControlQuestion{
		{ID: "q01", Question: "Как проект получает HTTPS на VPS?", ExpectedSources: []string{"deploy/Caddyfile", "deploy/compose.yaml"}},
		{ID: "q02", Question: "Какие два MCP-сервера оркестрирует День 20?", ExpectedSources: []string{"challenges/day-20/README.md", "backend/internal/application/business_network.go"}},
		{ID: "q03", Question: "Где сохраняется история автоматических дайджестов?", ExpectedSources: []string{"backend/internal/infrastructure/sqlite/radar_store.go"}},
		{ID: "q04", Question: "Как запускается агент Дня 18 по расписанию?", ExpectedSources: []string{"challenges/day-18/README.md", "deploy/compose.yaml"}},
		{ID: "q05", Question: "Какие режимы ответа сравниваются во втором задании?", ExpectedSources: []string{"challenges/day-02/README.md"}},
		{ID: "q06", Question: "Какие фазы есть у состояния задачи агента?", ExpectedSources: []string{"backend/internal/domain/task_state.go", "challenges/day-13/README.md"}},
		{ID: "q07", Question: "Какие три стратегии контекста реализованы в Дне 10?", ExpectedSources: []string{"challenges/day-10/README.md"}},
		{ID: "q08", Question: "Какие инструменты входят в Research MCP server?", ExpectedSources: []string{"backend/internal/infrastructure/mcpserver/research.go", "challenges/day-19/README.md"}},
		{ID: "q09", Question: "Как web-приложение переключает лаборатории разных дней?", ExpectedSources: []string{"web/src/App.tsx"}},
		{ID: "q10", Question: "Почему API-ключ модели не попадает в браузер?", ExpectedSources: []string{"deploy/Caddyfile", "README.md"}},
	}
}

func (s *RAGService) Compare(ctx context.Context, question string) (domain.RAGComparison, error) {
	normalized, err := s.validateQuestion(question)
	if err != nil {
		return domain.RAGComparison{}, err
	}
	retrieved, err := s.Retrieve(ctx, normalized, 5)
	if err != nil {
		return domain.RAGComparison{}, err
	}
	without, err := s.generate(ctx, `Ты отвечаешь на вопрос пользователя без доступа к базе знаний проекта. Не выдумывай детали: если точных данных нет, честно скажи об этом. Отвечай кратко на языке пользователя.`, normalized)
	if err != nil {
		return domain.RAGComparison{}, fmt.Errorf("answer without RAG: %w", err)
	}
	contextBlock := formatRetrievedContext(retrieved)
	with, err := s.generate(ctx, `Ты отвечаешь только по переданному контексту из репозитория AI Challenge. Сформулируй точный законченный ответ на языке пользователя. Не используй внешние знания и не придумывай отсутствующие факты. В конце назови пути использованных файлов.`, "Вопрос:\n"+normalized+"\n\nКонтекст:\n"+contextBlock)
	if err != nil {
		return domain.RAGComparison{}, fmt.Errorf("answer with RAG: %w", err)
	}
	return domain.RAGComparison{Question: normalized, WithoutRAG: without, WithRAG: with, Retrieved: retrieved}, nil
}

func (s *RAGService) Retrieve(ctx context.Context, question string, topK int) ([]domain.RetrievedChunk, error) {
	chunks, err := s.store.LoadKnowledgeChunks(ctx, domain.ChunkStrategyStructural)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		if _, err := s.indexer.Build(ctx); err != nil {
			return nil, fmt.Errorf("build missing knowledge index: %w", err)
		}
		chunks, err = s.store.LoadKnowledgeChunks(ctx, domain.ChunkStrategyStructural)
		if err != nil {
			return nil, err
		}
	}
	queryVector := embedText(question)
	results := make([]domain.RetrievedChunk, 0, len(chunks))
	for _, chunk := range chunks {
		results = append(results, domain.RetrievedChunk{Chunk: chunk, SimilarityScore: cosine(queryVector, chunk.Vector)})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].SimilarityScore > results[j].SimilarityScore })
	if topK < 1 {
		topK = 1
	}
	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

func (s *RAGService) validateQuestion(question string) (string, error) {
	normalized := strings.TrimSpace(question)
	if normalized == "" {
		return "", ErrEmptyRAGQuestion
	}
	if utf8.RuneCountInString(normalized) > s.maxRunes {
		return "", ErrRAGQuestionTooLong
	}
	return normalized, nil
}

func (s *RAGService) generate(ctx context.Context, system, user string) (domain.RAGAnswer, error) {
	started := time.Now()
	temperature := 0.1
	response, err := s.client.Generate(ctx, domain.ModelRequest{SystemPrompt: system, UserPrompt: user, Temperature: &temperature, MaxTokens: 900})
	if err != nil {
		return domain.RAGAnswer{}, err
	}
	return domain.RAGAnswer{Answer: strings.TrimSpace(response.Content), Model: response.Model, DurationMS: time.Since(started).Milliseconds(), Usage: response.Usage}, nil
}

func cosine(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	dot, leftNorm, rightNorm := 0.0, 0.0, 0.0
	for index := range left {
		dot += left[index] * right[index]
		leftNorm += left[index] * left[index]
		rightNorm += right[index] * right[index]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

func formatRetrievedContext(chunks []domain.RetrievedChunk) string {
	var builder strings.Builder
	for index, item := range chunks {
		fmt.Fprintf(&builder, "[%d] source=%s; section=%s; score=%.3f\n%s\n\n", index+1, item.Chunk.Source, item.Chunk.Section, item.SimilarityScore, item.Chunk.Content)
	}
	return strings.TrimSpace(builder.String())
}
