package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var expectedControlledAnswerFields = map[string]struct{}{
	"status":      {},
	"answer":      {},
	"ingredients": {},
	"steps":       {},
}

func validateAndNormalizeControlledAnswer(answer string) (string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(answer), &fields); err != nil {
		return "", fmt.Errorf("decode JSON object: %w", err)
	}
	if fields == nil || len(fields) != len(expectedControlledAnswerFields) {
		return "", errors.New("response must contain exactly status, answer, ingredients, and steps")
	}
	for field := range expectedControlledAnswerFields {
		if _, exists := fields[field]; !exists {
			return "", fmt.Errorf("required field %q is missing", field)
		}
	}

	var payload domain.ControlledAnswer
	if err := json.Unmarshal([]byte(answer), &payload); err != nil {
		return "", fmt.Errorf("decode controlled answer: %w", err)
	}
	if payload.Status != domain.ControlledAnswerStatusOK && payload.Status != domain.ControlledAnswerStatusOutOfScope {
		return "", fmt.Errorf("unsupported status %q", payload.Status)
	}

	payload.Answer = strings.TrimSpace(payload.Answer)
	if payload.Answer == "" {
		return "", errors.New("answer must not be empty")
	}
	if payload.Ingredients == nil || payload.Steps == nil {
		return "", errors.New("ingredients and steps must be JSON arrays")
	}
	if len(strings.Fields(payload.Answer)) > domain.ControlledAnswerSummaryLimit {
		return "", fmt.Errorf("answer exceeds %d words", domain.ControlledAnswerSummaryLimit)
	}
	if len(payload.Ingredients) > domain.ControlledAnswerIngredientLimit {
		return "", fmt.Errorf("ingredients exceed %d items", domain.ControlledAnswerIngredientLimit)
	}
	if len(payload.Steps) > domain.ControlledAnswerStepLimit {
		return "", fmt.Errorf("steps exceed %d items", domain.ControlledAnswerStepLimit)
	}

	var err error
	payload.Ingredients, err = normalizeNonEmptyStrings(payload.Ingredients, "ingredients")
	if err != nil {
		return "", err
	}
	payload.Steps, err = normalizeNonEmptyStrings(payload.Steps, "steps")
	if err != nil {
		return "", err
	}

	if payload.Status == domain.ControlledAnswerStatusOutOfScope {
		if payload.Answer != domain.FoodOutOfScopeMessage {
			return "", errors.New("out_of_scope answer must use the configured refusal")
		}
		if len(payload.Ingredients) != 0 || len(payload.Steps) != 0 {
			return "", errors.New("out_of_scope ingredients and steps must be empty")
		}
	}

	if controlledAnswerWordCount(payload) > domain.ControlledAnswerWordLimit {
		return "", fmt.Errorf("controlled answer exceeds %d words", domain.ControlledAnswerWordLimit)
	}

	normalized, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("normalize controlled answer: %w", err)
	}
	return string(normalized), nil
}

func normalizeNonEmptyStrings(values []string, field string) ([]string, error) {
	for index := range values {
		values[index] = strings.TrimSpace(values[index])
		if values[index] == "" {
			return nil, fmt.Errorf("%s must not contain empty strings", field)
		}
	}
	return values, nil
}

func controlledAnswerWordCount(payload domain.ControlledAnswer) int {
	count := len(strings.Fields(payload.Answer))
	for _, ingredient := range payload.Ingredients {
		count += len(strings.Fields(ingredient))
	}
	for _, step := range payload.Steps {
		count += len(strings.Fields(step))
	}
	return count
}
