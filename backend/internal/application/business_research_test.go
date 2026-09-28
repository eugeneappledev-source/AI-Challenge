package application_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/infrastructure/mcpserver"
)

type fakeNewsProvider struct{}

func (fakeNewsProvider) Search(_ context.Context, query string, _ int) ([]domain.BusinessStory, error) {
	return []domain.BusinessStory{{ID: "42", Title: "AI startup launches a new product", URL: "https://example.com/story", DiscussionURL: "https://news.ycombinator.com/item?id=42", Source: "Hacker News", Score: 120, PublishedAt: time.Unix(1_800_000_000, 0).UTC()}}, nil
}

type sequenceModel struct{ calls int }

func (m *sequenceModel) Generate(_ context.Context, request domain.ModelRequest) (domain.ModelResponse, error) {
	m.calls++
	if m.calls == 1 {
		return domain.ModelResponse{Content: `{"tool":"search_business_news","arguments":{"query":"AI startups","limit":5},"rationale":"Нужны свежие новости"}`, Model: "test-model", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 10}}, nil
	}
	return domain.ModelResponse{Content: `{"summary":"Рынок активен.","opportunities":[{"title":"Мини-сервис","whyNow":"Есть спрос","firstStep":"Проверить интервью","risk":"Нет платящих клиентов"}],"caveat":"Это гипотеза, а не гарантия дохода."}`, Model: "test-model", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 20}}, nil
}

func TestBusinessResearchAgentSelectsAndCallsRegisteredMCPTool(t *testing.T) {
	mcpHTTP := httptest.NewServer(mcpserver.StreamableHandler(mcpserver.NewResearchServer(fakeNewsProvider{})))
	defer mcpHTTP.Close()
	model := &sequenceModel{}
	service := application.NewBusinessResearchService(model, mcpHTTP.URL, 1000)

	result, err := service.Research(context.Background(), "Найди новости про AI-стартапы")
	if err != nil {
		t.Fatalf("Research() error = %v", err)
	}
	if result.Selection.Tool != "search_business_news" || len(result.Stories) != 1 {
		t.Fatalf("unexpected tool result: %+v", result)
	}
	if model.calls != 2 || result.Usage.TotalTokens != 30 || len(result.Advice.Opportunities) != 1 {
		t.Fatalf("unexpected agent result: %+v", result)
	}
}
