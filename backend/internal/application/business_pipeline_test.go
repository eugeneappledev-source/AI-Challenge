package application_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/infrastructure/mcpserver"
)

type memoryReportStore struct{ reports []domain.RadarReport }

func (s *memoryReportStore) SaveRadarReport(_ context.Context, report domain.RadarReport) error {
	s.reports = append(s.reports, report)
	return nil
}

type pipelineModel struct{ calls int }

func (m *pipelineModel) Generate(_ context.Context, _ domain.ModelRequest) (domain.ModelResponse, error) {
	m.calls++
	if m.calls == 1 {
		return domain.ModelResponse{Content: `{"tools":["search_business_news","summarize_business_news","save_business_report"],"searchArguments":{"query":"AI startup","limit":3},"rationale":"Сначала факты, затем анализ и сохранение"}`, Model: "test", Usage: domain.Usage{TotalTokens: 10}}, nil
	}
	return domain.ModelResponse{Content: `{"summary":"Есть новый сигнал.","keySignals":["Запуск продукта"],"risks":["Спрос не подтвержден"],"nextQuestion":"Кто заплатит?"}`, Model: "test", Usage: domain.Usage{TotalTokens: 20}}, nil
}

func TestBusinessPipelineChainsThreeToolsAndPersistsReport(t *testing.T) {
	model := &pipelineModel{}
	store := &memoryReportStore{}
	server := mcpserver.NewResearchPipelineServer(fakeNewsProvider{}, application.NewLLMNewsSummarizer(model), store)
	httpServer := httptest.NewServer(mcpserver.StreamableHandler(server))
	defer httpServer.Close()
	service := application.NewBusinessPipelineService(model, httpServer.URL)
	result, err := service.Run(context.Background(), "Исследуй AI-стартапы")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Stages) != 3 || len(store.reports) != 1 {
		t.Fatalf("unexpected pipeline result: %+v", result)
	}
	if result.Stages[1].InputSummary != "handoff: 1 stories" || result.SavedReport.ID == "" {
		t.Fatalf("handoff or persistence missing: %+v", result)
	}
	if model.calls != 2 || result.Usage.TotalTokens != 30 {
		t.Fatalf("unexpected model usage: %+v", result.Usage)
	}
}
