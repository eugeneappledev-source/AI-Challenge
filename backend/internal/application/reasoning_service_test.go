package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type reasoningClientStub struct {
	requests  []domain.ModelRequest
	responses []domain.ModelResponse
	err       error
}

func (s *reasoningClientStub) Generate(_ context.Context, request domain.ModelRequest) (domain.ModelResponse, error) {
	s.requests = append(s.requests, request)
	if s.err != nil {
		return domain.ModelResponse{}, s.err
	}
	if len(s.responses) == 0 {
		return domain.ModelResponse{}, errors.New("missing stub response")
	}
	response := s.responses[0]
	s.responses = s.responses[1:]
	return response, nil
}

func TestReasoningDirectForwardsProblemWithoutAdditionalUserInstructions(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{{Content: "42", Model: "model", FinishReason: "stop"}}}
	service := NewReasoningService(client, 1000)

	attempt, err := service.Run(context.Background(), "  Сколько будет 40 + 2?  ", domain.ReasoningMethodDirect)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.requests) != 1 || client.requests[0].UserPrompt != "Сколько будет 40 + 2?" {
		t.Fatalf("expected the normalized problem only, got %+v", client.requests)
	}
	if client.requests[0].JSON {
		t.Fatal("direct method must not request structured output")
	}
	if attempt.Method != domain.ReasoningMethodDirect || attempt.Answer != "42" {
		t.Fatalf("unexpected attempt: %+v", attempt)
	}
}

func TestReasoningStepByStepAddsExplicitInstruction(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{{Content: "Шаги", FinishReason: "stop"}}}
	service := NewReasoningService(client, 1000)

	_, err := service.Run(context.Background(), "Логическая задача", domain.ReasoningMethodStepByStep)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prompt := client.requests[0].UserPrompt
	if !strings.Contains(prompt, "Логическая задача") || !strings.Contains(prompt, "Решай пошагово") {
		t.Fatalf("expected problem and step-by-step instruction, got %q", prompt)
	}
}

func TestReasoningMetaPromptUsesGeneratedPromptForSecondCall(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{
		{Content: "Улучшенный промпт", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 11}},
		{Content: "Итоговое решение", Model: "model", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 17}},
	}}
	service := NewReasoningService(client, 1000)

	attempt, err := service.Run(context.Background(), "Задача", domain.ReasoningMethodMetaPrompt)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.requests) != 2 {
		t.Fatalf("expected two model calls, got %d", len(client.requests))
	}
	if client.requests[1].UserPrompt != "Улучшенный промпт" {
		t.Fatalf("expected generated prompt in second call, got %q", client.requests[1].UserPrompt)
	}
	if attempt.GeneratedPrompt != "Улучшенный промпт" || attempt.Answer != "Итоговое решение" {
		t.Fatalf("unexpected attempt: %+v", attempt)
	}
	if attempt.Usage.TotalTokens != 28 {
		t.Fatalf("expected summed usage, got %+v", attempt.Usage)
	}
}

func TestReasoningExpertPanelReturnsEachExpertAndSynthesis(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{{
		Content: `{"experts":[{"role":"Аналитик","answer":"A"},{"role":"Инженер","answer":"B"},{"role":"Критик","answer":"C"}],"synthesis":"Итог"}`,
		Model:   "model", FinishReason: "stop",
	}}}
	service := NewReasoningService(client, 1000)

	attempt, err := service.Run(context.Background(), "Задача", domain.ReasoningMethodExpertPanel)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !client.requests[0].JSON {
		t.Fatal("expert panel must request JSON")
	}
	if len(attempt.Experts) != 3 || attempt.Answer != "Итог" {
		t.Fatalf("unexpected expert attempt: %+v", attempt)
	}
}

func TestReasoningReviewMapsAnonymousWinnerBackToMethod(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{{
		Content: `{
			"winner":"A",
			"verdict":"Промпт помог проверить ответ.",
			"referenceAnswer":"Эталон",
			"differences":["Разная глубина проверки"],
			"scores":[
				{"label":"A","correctness":10,"clarity":9,"verification":10,"feedback":"Верно"},
				{"label":"B","correctness":8,"clarity":8,"verification":6,"feedback":"Мало проверки"},
				{"label":"C","correctness":9,"clarity":8,"verification":9,"feedback":"Хорошо"},
				{"label":"D","correctness":9,"clarity":9,"verification":8,"feedback":"Верно"}
			]
		}`,
		Model: "review-model",
		Usage: domain.Usage{TotalTokens: 50},
	}}}
	service := NewReasoningService(client, 1000)

	review, err := service.Review(context.Background(), "Задача", completeAttempts())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Winner != domain.ReasoningMethodMetaPrompt {
		t.Fatalf("expected label A to map to meta_prompt, got %q", review.Winner)
	}
	if len(review.Scores) != 4 || review.Model != "review-model" {
		t.Fatalf("unexpected review: %+v", review)
	}
	prompt := client.requests[0].UserPrompt
	if strings.Contains(prompt, `"method"`) || !strings.Contains(prompt, `"label":"A"`) {
		t.Fatalf("expected anonymized labeled attempts, got %q", prompt)
	}
}

func TestReasoningReviewRequiresEveryMethodExactlyOnce(t *testing.T) {
	service := NewReasoningService(&reasoningClientStub{}, 1000)
	attempts := completeAttempts()
	attempts[3].Method = domain.ReasoningMethodDirect

	_, err := service.Review(context.Background(), "Задача", attempts)

	if !errors.Is(err, ErrInvalidReasoningAttempts) {
		t.Fatalf("expected ErrInvalidReasoningAttempts, got %v", err)
	}
}

func TestReasoningReviewRepairsInvalidJSONOnce(t *testing.T) {
	client := &reasoningClientStub{responses: []domain.ModelResponse{
		{Content: "not valid json", Model: "model", Usage: domain.Usage{TotalTokens: 20}},
		{
			Content: `{
				"winner":"B",
				"verdict":"Исправлено",
				"referenceAnswer":"Эталон",
				"differences":["Отличие"],
				"scores":[
					{"label":"A","correctness":8,"clarity":8,"verification":8,"feedback":"A"},
					{"label":"B","correctness":10,"clarity":9,"verification":10,"feedback":"B"},
					{"label":"C","correctness":8,"clarity":8,"verification":8,"feedback":"C"},
					{"label":"D","correctness":8,"clarity":8,"verification":8,"feedback":"D"}
				]
			}`,
			Model: "model", Usage: domain.Usage{TotalTokens: 30},
		},
	}}
	service := NewReasoningService(client, 1000)

	review, err := service.Review(context.Background(), "Задача", completeAttempts())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.requests) != 2 {
		t.Fatalf("expected one repair call, got %d requests", len(client.requests))
	}
	if review.Winner != domain.ReasoningMethodDirect || review.Usage.TotalTokens != 50 {
		t.Fatalf("unexpected repaired review: %+v", review)
	}
}

func completeAttempts() []domain.ReasoningAttempt {
	return []domain.ReasoningAttempt{
		{Method: domain.ReasoningMethodDirect, Answer: "direct"},
		{Method: domain.ReasoningMethodStepByStep, Answer: "steps"},
		{Method: domain.ReasoningMethodMetaPrompt, Answer: "meta", GeneratedPrompt: "prompt"},
		{Method: domain.ReasoningMethodExpertPanel, Answer: "experts", Experts: []domain.ExpertSolution{{Role: "Аналитик", Answer: "A"}}},
	}
}
