package application_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/infrastructure/mcpserver"
)

type networkModel struct{ calls int }

func (m *networkModel) Generate(_ context.Context, _ domain.ModelRequest) (domain.ModelResponse, error) {
	m.calls++
	responses := []string{
		`{"route":[{"server":"startup-research-mcp","tool":"search_business_news"},{"server":"startup-research-mcp","tool":"summarize_business_news"},{"server":"business-advisor-mcp","tool":"score_business_opportunities"},{"server":"business-advisor-mcp","tool":"build_action_plan"},{"server":"startup-research-mcp","tool":"save_business_report"}],"rationale":"Research передаёт факты Advisor, затем сохраняет результат"}`,
		`{"summary":"Сигнал подтверждён источником.","keySignals":["AI automation"],"risks":["Неизвестный спрос"],"nextQuestion":"Кому это нужно?"}`,
		`{"items":[{"title":"AI помощник","evidence":"AI automation","fitScore":88,"effort":"low","rationale":"Подходит навыкам","risk":"Нет спроса"}]}`,
		`{"selectedTitle":"AI помощник","goal":"Получить 3 интервью","steps":["Найти 10 клиентов","Провести интервью","Собрать лендинг","Проверить заявку"],"stopConditions":["Нет проблемы после 5 интервью"],"budgetNote":"Не больше 100 USD"}`,
	}
	return domain.ModelResponse{Content: responses[m.calls-1], Model: "test", Usage: domain.Usage{TotalTokens: 10}}, nil
}

func TestBusinessNetworkRoutesAcrossTwoMCPServers(t *testing.T) {
	model := &networkModel{}
	store := &memoryReportStore{}
	researchHTTP := httptest.NewServer(mcpserver.StreamableHandler(mcpserver.NewResearchPipelineServer(fakeNewsProvider{}, application.NewLLMNewsSummarizer(model), store)))
	defer researchHTTP.Close()
	advisorHTTP := httptest.NewServer(mcpserver.StreamableHandler(mcpserver.NewAdvisorServer(application.NewLLMOpportunityAdvisor(model))))
	defer advisorHTTP.Close()
	service := application.NewBusinessNetworkService(model, researchHTTP.URL, advisorHTTP.URL)
	result, err := service.Run(context.Background(), "Найди идею", domain.FounderProfile{BudgetUSD: 100, HoursPerWeek: 8, Skills: []string{"iOS"}, RiskLevel: "low"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Servers) != 2 || len(result.Route) != 5 || len(store.reports) != 1 {
		t.Fatalf("unexpected network result: %+v", result)
	}
	if result.Route[1].Server == result.Route[2].Server {
		t.Fatalf("expected cross-server handoff: %+v", result.Route)
	}
	if model.calls != 4 || result.Usage.TotalTokens != 40 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
	}
}
