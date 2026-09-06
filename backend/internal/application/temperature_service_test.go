package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func TestTemperatureRunForwardsExactPromptAndTemperature(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{{
		Content: "Ответ", Model: "model", FinishReason: "stop",
	}}}
	service := NewTemperatureService(client, 1000)

	attempt, err := service.Run(context.Background(), "  Один запрос  ", domain.TemperaturePrecise)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.requests) != 1 || client.requests[0].UserPrompt != "Один запрос" {
		t.Fatalf("expected normalized prompt, got %+v", client.requests)
	}
	if client.requests[0].Temperature == nil || *client.requests[0].Temperature != 0 {
		t.Fatalf("expected explicit zero temperature, got %+v", client.requests[0].Temperature)
	}
	if attempt.Temperature != domain.TemperaturePrecise || attempt.Answer != "Ответ" {
		t.Fatalf("unexpected attempt: %+v", attempt)
	}
}

func TestTemperatureRunRejectsValueOutsideExperiment(t *testing.T) {
	service := NewTemperatureService(&reasoningClientStub{}, 1000)

	_, err := service.Run(context.Background(), "Запрос", domain.Temperature(0.5))

	if !errors.Is(err, ErrInvalidTemperature) {
		t.Fatalf("expected ErrInvalidTemperature, got %v", err)
	}
}

func TestTemperatureReviewMapsAnonymousLabelsToValues(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{{
		Content: `{
			"summary":"Чем выше температура, тем свободнее формулировки.",
			"bestAccuracy":"B",
			"bestCreativity":"A",
			"bestDiversity":"A",
			"differences":["B точнее, A образнее"],
			"scores":[
				{"label":"A","accuracy":8,"creativity":10,"diversity":10,"feedback":"Самый оригинальный"},
				{"label":"B","accuracy":10,"creativity":5,"diversity":4,"feedback":"Самый строгий"},
				{"label":"C","accuracy":9,"creativity":8,"diversity":7,"feedback":"Баланс"}
			],
			"recommendations":[
				{"label":"A","bestFor":["Идеи"],"caution":"Проверять факты"},
				{"label":"B","bestFor":["Факты"],"caution":"Меньше вариативности"},
				{"label":"C","bestFor":["Тексты"],"caution":"Не максимизирует крайности"}
			]
		}`,
		Model: "review-model", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 40},
	}}}
	service := NewTemperatureService(client, 1000)

	review, err := service.Review(context.Background(), "Один запрос", completeTemperatureAttempts())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.BestAccuracy != domain.TemperaturePrecise ||
		review.BestCreativity != domain.TemperatureCreative ||
		review.BestDiversity != domain.TemperatureCreative {
		t.Fatalf("unexpected best values: %+v", review)
	}
	if len(review.Scores) != 3 || len(review.Recommendations) != 3 {
		t.Fatalf("expected full feedback, got %+v", review)
	}
	if strings.Contains(client.requests[0].UserPrompt, `"temperature"`) || !strings.Contains(client.requests[0].UserPrompt, `"label":"A"`) {
		t.Fatalf("expected anonymized answers, got %q", client.requests[0].UserPrompt)
	}
	if client.requests[0].Temperature == nil || *client.requests[0].Temperature != 0 {
		t.Fatal("reviewer must use deterministic temperature 0")
	}
}

func TestTemperatureReviewRequiresAllThreeValues(t *testing.T) {
	service := NewTemperatureService(&reasoningClientStub{}, 1000)
	attempts := completeTemperatureAttempts()
	attempts[2].Temperature = domain.TemperatureBalanced

	_, err := service.Review(context.Background(), "Запрос", attempts)

	if !errors.Is(err, ErrInvalidTemperatureRuns) {
		t.Fatalf("expected ErrInvalidTemperatureRuns, got %v", err)
	}
}

func TestTemperatureReviewRepairsInvalidJSONOnce(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{
		{Content: "broken", Model: "model", Usage: domain.Usage{TotalTokens: 10}},
		{
			Content: `{
				"summary":"Исправлено","bestAccuracy":"B","bestCreativity":"C","bestDiversity":"A",
				"differences":["Разные формулировки"],
				"scores":[
					{"label":"A","accuracy":7,"creativity":10,"diversity":10,"feedback":"A"},
					{"label":"B","accuracy":10,"creativity":5,"diversity":5,"feedback":"B"},
					{"label":"C","accuracy":9,"creativity":8,"diversity":8,"feedback":"C"}
				],
				"recommendations":[
					{"label":"A","bestFor":["A"],"caution":"A"},
					{"label":"B","bestFor":["B"],"caution":"B"},
					{"label":"C","bestFor":["C"],"caution":"C"}
				]
			}`,
			Model: "model", Usage: domain.Usage{TotalTokens: 20},
		},
	}}
	service := NewTemperatureService(client, 1000)

	review, err := service.Review(context.Background(), "Запрос", completeTemperatureAttempts())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.requests) != 2 || review.Usage.TotalTokens != 30 {
		t.Fatalf("expected one repair and summed usage, got requests=%d review=%+v", len(client.requests), review)
	}
}

func completeTemperatureAttempts() []domain.TemperatureAttempt {
	return []domain.TemperatureAttempt{
		{Temperature: domain.TemperaturePrecise, Answer: "precise"},
		{Temperature: domain.TemperatureBalanced, Answer: "balanced"},
		{Temperature: domain.TemperatureCreative, Answer: "creative"},
	}
}
