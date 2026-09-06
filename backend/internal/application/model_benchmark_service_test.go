package application

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func TestModelBenchmarkRunChangesOnlyModelProfile(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{{
		Content: "Ответ", Model: "deepseek-v4-pro", FinishReason: "stop",
		Usage: domain.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, PromptCacheMissTokens: 100},
	}}}
	service := NewModelBenchmarkService(client, 1000)
	times := []time.Time{
		time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 7, 12, 0, 1, 250_000_000, time.UTC),
	}
	service.now = func() time.Time {
		value := times[0]
		times = times[1:]
		return value
	}

	attempt, err := service.Run(context.Background(), "  Один запрос  ", domain.ModelTierStrong)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	request := client.requests[0]
	if request.Model != "deepseek-v4-pro" || request.UserPrompt != "Один запрос" {
		t.Fatalf("unexpected request: %+v", request)
	}
	if request.Temperature == nil || *request.Temperature != 0 || request.MaxTokens != modelBenchmarkMaxTokens {
		t.Fatalf("expected fixed experiment controls, got %+v", request)
	}
	if attempt.LatencyMilliseconds != 1250 || attempt.PricingPeriod != "off_peak" {
		t.Fatalf("unexpected measurements: %+v", attempt)
	}
	expected := (100*0.66 + 50*1.98) / 1_000_000
	if math.Abs(attempt.EstimatedCostUSD-expected) > 1e-12 {
		t.Fatalf("expected cost %.12f, got %.12f", expected, attempt.EstimatedCostUSD)
	}
}

func TestModelBenchmarkCostUsesCacheAndPeakPricing(t *testing.T) {
	usage := domain.Usage{
		PromptTokens: 1000, CompletionTokens: 100,
		PromptCacheHitTokens: 600, PromptCacheMissTokens: 300,
	}
	input, output := calculateModelCost(usage, modelProfiles[domain.ModelTierBasic].peak)
	expectedInput := (600*0.014 + 400*0.44) / 1_000_000
	expectedOutput := 100 * 1.32 / 1_000_000
	if math.Abs(input-expectedInput) > 1e-12 || math.Abs(output-expectedOutput) > 1e-12 {
		t.Fatalf("unexpected costs input=%f output=%f", input, output)
	}
	if !isDeepSeekPeak(time.Date(2026, 9, 7, 6, 30, 0, 0, time.UTC)) {
		t.Fatal("expected weekday 06:30 UTC to be peak")
	}
	if isDeepSeekPeak(time.Date(2026, 9, 6, 6, 30, 0, 0, time.UTC)) {
		t.Fatal("weekend must be off-peak")
	}
}

func TestModelBenchmarkRunRejectsUnknownTier(t *testing.T) {
	service := NewModelBenchmarkService(&reasoningClientStub{}, 1000)
	_, err := service.Run(context.Background(), "Запрос", domain.ModelTier("medium"))
	if !errors.Is(err, ErrInvalidModelTier) {
		t.Fatalf("expected ErrInvalidModelTier, got %v", err)
	}
}

func TestModelBenchmarkReviewIsAnonymousAndMapsLabels(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{{
		Content: `{
			"qualityWinner":"A","summary":"Pro дал наиболее полный ответ.",
			"differences":["A точнее соблюдает ограничения"],
			"scores":[
				{"label":"A","accuracy":10,"completeness":9,"clarity":9,"feedback":"Сильный ответ"},
				{"label":"B","accuracy":8,"completeness":7,"clarity":9,"feedback":"Кратко"},
				{"label":"C","accuracy":9,"completeness":8,"clarity":8,"feedback":"Полезно"}
			],
			"recommendations":[
				{"label":"A","bestFor":["Сложный анализ"],"tradeoff":"Выше стоимость"},
				{"label":"B","bestFor":["Простые ответы"],"tradeoff":"Меньше глубины"},
				{"label":"C","bestFor":["Мультимодальные задачи"],"tradeoff":"Экспериментальная версия"}
			]
		}`,
		Model: "deepseek-v4-pro", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 80},
	}}}
	service := NewModelBenchmarkService(client, 1000)
	attempts := completeModelAttempts()

	review, err := service.Review(context.Background(), "Один запрос", attempts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.QualityWinner != domain.ModelTierStrong || review.Fastest != domain.ModelTierExtended || review.Cheapest != domain.ModelTierBasic {
		t.Fatalf("unexpected winners: %+v", review)
	}
	prompt := client.requests[0].UserPrompt
	if strings.Contains(prompt, "deepseek-v4") || strings.Contains(prompt, `"tier"`) || !strings.Contains(prompt, `"label":"A"`) {
		t.Fatalf("review must receive anonymous content only: %s", prompt)
	}
	if client.requests[0].Model != modelReviewerID || client.requests[0].Temperature == nil || *client.requests[0].Temperature != 0 {
		t.Fatalf("unexpected reviewer request: %+v", client.requests[0])
	}
}

func TestModelBenchmarkReviewRequiresEveryTier(t *testing.T) {
	service := NewModelBenchmarkService(&reasoningClientStub{}, 1000)
	attempts := completeModelAttempts()
	attempts[2].Tier = domain.ModelTierBasic
	_, err := service.Review(context.Background(), "Запрос", attempts)
	if !errors.Is(err, ErrInvalidModelRuns) {
		t.Fatalf("expected ErrInvalidModelRuns, got %v", err)
	}
}

func TestModelBenchmarkReviewRepairsInvalidJSONOnce(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{
		{Content: "broken", Model: modelReviewerID, Usage: domain.Usage{TotalTokens: 10}},
		{Content: `{
			"qualityWinner":"B","summary":"Исправлено","differences":["Различия есть"],
			"scores":[
				{"label":"A","accuracy":8,"completeness":8,"clarity":8,"feedback":"A"},
				{"label":"B","accuracy":9,"completeness":9,"clarity":9,"feedback":"B"},
				{"label":"C","accuracy":7,"completeness":7,"clarity":7,"feedback":"C"}
			],
			"recommendations":[
				{"label":"A","bestFor":["A"],"tradeoff":"A"},
				{"label":"B","bestFor":["B"],"tradeoff":"B"},
				{"label":"C","bestFor":["C"],"tradeoff":"C"}
			]
		}`, Model: modelReviewerID, Usage: domain.Usage{TotalTokens: 20}},
	}}
	service := NewModelBenchmarkService(client, 1000)

	review, err := service.Review(context.Background(), "Запрос", completeModelAttempts())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.requests) != 2 || review.ReviewerUsage.TotalTokens != 30 {
		t.Fatalf("expected one repair and summed usage, got requests=%d review=%+v", len(client.requests), review)
	}
}

func completeModelAttempts() []domain.ModelBenchmarkAttempt {
	return []domain.ModelBenchmarkAttempt{
		{Tier: domain.ModelTierBasic, Answer: "basic", LatencyMilliseconds: 900, EstimatedCostUSD: 0.0001},
		{Tier: domain.ModelTierExtended, Answer: "extended", LatencyMilliseconds: 700, EstimatedCostUSD: 0.0002},
		{Tier: domain.ModelTierStrong, Answer: "strong", LatencyMilliseconds: 1200, EstimatedCostUSD: 0.0004},
	}
}
