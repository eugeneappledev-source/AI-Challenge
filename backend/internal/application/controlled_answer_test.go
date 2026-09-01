package application

import (
	"strings"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func TestValidateAndNormalizeControlledAnswer(t *testing.T) {
	answer := `{"status":"out_of_scope","answer":"` + domain.FoodOutOfScopeMessage + `","ingredients":[],"steps":[]}`

	normalized, err := validateAndNormalizeControlledAnswer(answer)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{
  "status": "out_of_scope",
  "answer": "Я отвечаю только на вопросы о еде и приготовлении. Пожалуйста, задайте вопрос по теме.",
  "ingredients": [],
  "steps": []
}`
	if normalized != expected {
		t.Fatalf("unexpected normalized answer:\n%s", normalized)
	}
}

func TestValidateControlledAnswerRejectsContractViolations(t *testing.T) {
	tooLongSummary := strings.TrimSpace(strings.Repeat("слово ", domain.ControlledAnswerSummaryLimit+1))
	longStep := strings.TrimSpace(strings.Repeat("слово ", 20))
	tests := []struct {
		name   string
		answer string
	}{
		{name: "not JSON", answer: `not-json`},
		{name: "missing field", answer: `{"status":"ok","answer":"Да","ingredients":[]}`},
		{name: "extra field", answer: `{"status":"ok","answer":"Да","ingredients":[],"steps":[],"extra":true}`},
		{name: "invalid status", answer: `{"status":"unknown","answer":"Да","ingredients":[],"steps":[]}`},
		{name: "null arrays", answer: `{"status":"ok","answer":"Да","ingredients":null,"steps":null}`},
		{name: "empty array item", answer: `{"status":"ok","answer":"Да","ingredients":[""],"steps":[]}`},
		{name: "wrong refusal", answer: `{"status":"out_of_scope","answer":"Не знаю","ingredients":[],"steps":[]}`},
		{name: "out of scope content", answer: `{"status":"out_of_scope","answer":"` + domain.FoodOutOfScopeMessage + `","ingredients":["сыр"],"steps":[]}`},
		{name: "summary limit", answer: `{"status":"ok","answer":"` + tooLongSummary + `","ingredients":[],"steps":[]}`},
		{name: "ingredient limit", answer: `{"status":"ok","answer":"Да","ingredients":["1","2","3","4","5","6","7","8","9"],"steps":[]}`},
		{name: "step limit", answer: `{"status":"ok","answer":"Да","ingredients":[],"steps":["1","2","3","4","5"]}`},
		{name: "word limit", answer: `{"status":"ok","answer":"Да","ingredients":[],"steps":["` + longStep + `","` + longStep + `","` + longStep + `","` + longStep + `"]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := validateAndNormalizeControlledAnswer(test.answer); err == nil {
				t.Fatalf("expected contract violation for %s", test.answer)
			}
		})
	}
}
