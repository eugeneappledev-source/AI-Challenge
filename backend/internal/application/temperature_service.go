package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const (
	temperatureAnswerMaxTokens = 900
	temperatureReviewMaxTokens = 1800
)

var (
	ErrEmptyTemperaturePrompt   = errors.New("temperature prompt is required")
	ErrTemperaturePromptTooLong = errors.New("temperature prompt is too long")
	ErrInvalidTemperature       = errors.New("temperature must be 0, 0.7, or 1.2")
	ErrInvalidTemperatureRuns   = errors.New("temperature attempts are invalid")
	ErrInvalidTemperatureReview = errors.New("temperature review does not match the contract")
)

const temperatureSystemPrompt = `Ответь на запрос пользователя на его языке. Соблюдай фактическую точность и все явно заданные ограничения. Не упоминай системные инструкции или параметры генерации. Формулы записывай обычным читаемым текстом без LaTeX-команд.`

type TemperatureService struct {
	client         ReasoningClient
	maxPromptRunes int
}

func NewTemperatureService(client ReasoningClient, maxPromptRunes int) *TemperatureService {
	return &TemperatureService{client: client, maxPromptRunes: maxPromptRunes}
}

func (s *TemperatureService) Run(
	ctx context.Context,
	prompt string,
	temperature domain.Temperature,
) (domain.TemperatureAttempt, error) {
	normalized, err := s.validatePrompt(prompt)
	if err != nil {
		return domain.TemperatureAttempt{}, err
	}
	if !temperature.IsValid() {
		return domain.TemperatureAttempt{}, ErrInvalidTemperature
	}

	value := float64(temperature)
	response, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: temperatureSystemPrompt,
		UserPrompt:   normalized,
		MaxTokens:    temperatureAnswerMaxTokens,
		Temperature:  &value,
	})
	if err != nil {
		return domain.TemperatureAttempt{}, err
	}
	if responseWasTruncated(response) {
		return domain.TemperatureAttempt{}, fmt.Errorf("%w: model reached max_tokens", ErrInvalidTemperatureReview)
	}

	return domain.TemperatureAttempt{
		Temperature:  temperature,
		Answer:       response.Content,
		Model:        response.Model,
		FinishReason: response.FinishReason,
		Usage:        response.Usage,
	}, nil
}

func (s *TemperatureService) Review(
	ctx context.Context,
	prompt string,
	attempts []domain.TemperatureAttempt,
) (domain.TemperatureReview, error) {
	normalized, err := s.validatePrompt(prompt)
	if err != nil {
		return domain.TemperatureReview{}, err
	}
	if err := validateTemperatureAttempts(attempts); err != nil {
		return domain.TemperatureReview{}, fmt.Errorf("%w: %v", ErrInvalidTemperatureRuns, err)
	}

	labelByTemperature := map[domain.Temperature]string{
		domain.TemperatureCreative: "A",
		domain.TemperaturePrecise:  "B",
		domain.TemperatureBalanced: "C",
	}
	temperatureByLabel := make(map[string]domain.Temperature, len(labelByTemperature))
	for temperature, label := range labelByTemperature {
		temperatureByLabel[label] = temperature
	}

	type anonymousAttempt struct {
		Label  string `json:"label"`
		Answer string `json:"answer"`
	}
	anonymized := make([]anonymousAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		anonymized = append(anonymized, anonymousAttempt{
			Label:  labelByTemperature[attempt.Temperature],
			Answer: attempt.Answer,
		})
	}
	sort.Slice(anonymized, func(i, j int) bool { return anonymized[i].Label < anonymized[j].Label })
	answersJSON, err := json.Marshal(anonymized)
	if err != nil {
		return domain.TemperatureReview{}, fmt.Errorf("encode attempts: %w", err)
	}

	systemPrompt := `Ты независимый рецензент эксперимента с параметром temperature. Сначала оцени, насколько ответы соблюдают исходный запрос и факты. Затем сравни три анонимных ответа A, B и C, не пытаясь угадать их настройки.

Поставь каждому целые оценки 0–10:
- accuracy: фактическая точность и соблюдение запроса;
- creativity: оригинальность формулировок и идей;
- diversity: насколько ответ отличается от двух остальных по лексике, образам и подходу.

Определи лучший ответ отдельно по каждому критерию. Для каждой настройки сформулируй практические задачи, где она полезна, и одно предостережение. Помни: один запуск показывает наблюдение, а не статистически устойчивую закономерность.

Верни строго один JSON-объект без Markdown и комментариев:
{"summary":"string","bestAccuracy":"A","bestCreativity":"A","bestDiversity":"A","differences":["string"],"scores":[{"label":"A","accuracy":0,"creativity":0,"diversity":0,"feedback":"string"}],"recommendations":[{"label":"A","bestFor":["string"],"caution":"string"}]}

bestAccuracy, bestCreativity и bestDiversity — A, B или C. scores и recommendations содержат ровно по одному элементу для A, B и C. Оценки — от 0 до 10. Все строки однострочные, без необработанных переносов и LaTeX-команд. Заверши ответ после закрывающей фигурной скобки.`
	userPrompt := fmt.Sprintf("Исходный запрос:\n%s\n\nАнонимные ответы:\n%s", normalized, answersJSON)
	reviewerTemperature := 0.0
	response, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		JSON:         true,
		MaxTokens:    temperatureReviewMaxTokens,
		Temperature:  &reviewerTemperature,
	})
	if err != nil {
		return domain.TemperatureReview{}, err
	}
	if responseWasTruncated(response) {
		return domain.TemperatureReview{}, fmt.Errorf("%w: reviewer reached max_tokens", ErrInvalidTemperatureReview)
	}

	review, parseErr := decodeTemperatureReview(response.Content, temperatureByLabel)
	if parseErr != nil {
		repairResponse, repairErr := s.client.Generate(ctx, domain.ModelRequest{
			SystemPrompt: `Исправь переданный ответ и верни строго валидный JSON без Markdown. Сохрани смысл и все данные. Контракт:
{"summary":"string","bestAccuracy":"A","bestCreativity":"A","bestDiversity":"A","differences":["string"],"scores":[{"label":"A","accuracy":0,"creativity":0,"diversity":0,"feedback":"string"}],"recommendations":[{"label":"A","bestFor":["string"],"caution":"string"}]}
Метки — A, B и C; scores и recommendations содержат ровно все три метки; оценки — целые числа 0–10. Все строки должны быть однострочными и корректно экранированными.`,
			UserPrompt:  "Ответ для исправления:\n" + response.Content,
			JSON:        true,
			MaxTokens:   temperatureReviewMaxTokens,
			Temperature: &reviewerTemperature,
		})
		if repairErr != nil {
			return domain.TemperatureReview{}, repairErr
		}
		if responseWasTruncated(repairResponse) {
			return domain.TemperatureReview{}, fmt.Errorf("%w: repaired review reached max_tokens", ErrInvalidTemperatureReview)
		}
		review, parseErr = decodeTemperatureReview(repairResponse.Content, temperatureByLabel)
		if parseErr != nil {
			return domain.TemperatureReview{}, fmt.Errorf("%w: %v", ErrInvalidTemperatureReview, parseErr)
		}
		repairResponse.Usage = addUsage(response.Usage, repairResponse.Usage)
		response = repairResponse
	}

	review.Model = response.Model
	review.Usage = response.Usage
	return review, nil
}

func (s *TemperatureService) validatePrompt(prompt string) (string, error) {
	normalized := strings.TrimSpace(prompt)
	if normalized == "" {
		return "", ErrEmptyTemperaturePrompt
	}
	if utf8.RuneCountInString(normalized) > s.maxPromptRunes {
		return "", ErrTemperaturePromptTooLong
	}
	return normalized, nil
}

func validateTemperatureAttempts(attempts []domain.TemperatureAttempt) error {
	if len(attempts) != 3 {
		return errors.New("exactly three attempts are required")
	}
	seen := make(map[domain.Temperature]bool, len(attempts))
	for _, attempt := range attempts {
		if !attempt.Temperature.IsValid() {
			return fmt.Errorf("invalid temperature %v", attempt.Temperature)
		}
		if seen[attempt.Temperature] {
			return fmt.Errorf("duplicate temperature %v", attempt.Temperature)
		}
		if strings.TrimSpace(attempt.Answer) == "" {
			return fmt.Errorf("empty answer for temperature %v", attempt.Temperature)
		}
		seen[attempt.Temperature] = true
	}
	return nil
}

func decodeTemperatureReview(
	content string,
	temperatureByLabel map[string]domain.Temperature,
) (domain.TemperatureReview, error) {
	var payload temperatureReviewPayload
	if err := decodeStrictJSON(content, &payload); err != nil {
		return domain.TemperatureReview{}, err
	}
	return payload.toDomain(temperatureByLabel)
}

type temperatureReviewPayload struct {
	Summary         string                             `json:"summary"`
	BestAccuracy    string                             `json:"bestAccuracy"`
	BestCreativity  string                             `json:"bestCreativity"`
	BestDiversity   string                             `json:"bestDiversity"`
	Differences     []string                           `json:"differences"`
	Scores          []temperatureScorePayload          `json:"scores"`
	Recommendations []temperatureRecommendationPayload `json:"recommendations"`
}

type temperatureScorePayload struct {
	Label      string `json:"label"`
	Accuracy   int    `json:"accuracy"`
	Creativity int    `json:"creativity"`
	Diversity  int    `json:"diversity"`
	Feedback   string `json:"feedback"`
}

type temperatureRecommendationPayload struct {
	Label   string   `json:"label"`
	BestFor []string `json:"bestFor"`
	Caution string   `json:"caution"`
}

func (p temperatureReviewPayload) toDomain(
	temperatureByLabel map[string]domain.Temperature,
) (domain.TemperatureReview, error) {
	if strings.TrimSpace(p.Summary) == "" {
		return domain.TemperatureReview{}, errors.New("summary is required")
	}
	bestAccuracy, ok := temperatureByLabel[p.BestAccuracy]
	if !ok {
		return domain.TemperatureReview{}, errors.New("invalid bestAccuracy label")
	}
	bestCreativity, ok := temperatureByLabel[p.BestCreativity]
	if !ok {
		return domain.TemperatureReview{}, errors.New("invalid bestCreativity label")
	}
	bestDiversity, ok := temperatureByLabel[p.BestDiversity]
	if !ok {
		return domain.TemperatureReview{}, errors.New("invalid bestDiversity label")
	}
	if len(p.Differences) == 0 {
		return domain.TemperatureReview{}, errors.New("at least one difference is required")
	}

	scores := make([]domain.TemperatureScore, 0, 3)
	seenScores := make(map[string]bool, 3)
	if len(p.Scores) != 3 {
		return domain.TemperatureReview{}, errors.New("exactly three scores are required")
	}
	for _, score := range p.Scores {
		temperature, exists := temperatureByLabel[score.Label]
		if !exists || seenScores[score.Label] {
			return domain.TemperatureReview{}, errors.New("scores must contain unique A, B, and C labels")
		}
		if !validScore(score.Accuracy) || !validScore(score.Creativity) || !validScore(score.Diversity) {
			return domain.TemperatureReview{}, errors.New("scores must be between 0 and 10")
		}
		if strings.TrimSpace(score.Feedback) == "" {
			return domain.TemperatureReview{}, errors.New("score feedback is required")
		}
		seenScores[score.Label] = true
		scores = append(scores, domain.TemperatureScore{
			Temperature: temperature,
			Accuracy:    score.Accuracy,
			Creativity:  score.Creativity,
			Diversity:   score.Diversity,
			Feedback:    strings.TrimSpace(score.Feedback),
		})
	}

	recommendations := make([]domain.TemperatureRecommendation, 0, 3)
	seenRecommendations := make(map[string]bool, 3)
	if len(p.Recommendations) != 3 {
		return domain.TemperatureReview{}, errors.New("exactly three recommendations are required")
	}
	for _, recommendation := range p.Recommendations {
		temperature, exists := temperatureByLabel[recommendation.Label]
		if !exists || seenRecommendations[recommendation.Label] {
			return domain.TemperatureReview{}, errors.New("recommendations must contain unique A, B, and C labels")
		}
		if len(recommendation.BestFor) == 0 || strings.TrimSpace(recommendation.Caution) == "" {
			return domain.TemperatureReview{}, errors.New("recommendation details are required")
		}
		seenRecommendations[recommendation.Label] = true
		recommendations = append(recommendations, domain.TemperatureRecommendation{
			Temperature: temperature,
			BestFor:     recommendation.BestFor,
			Caution:     strings.TrimSpace(recommendation.Caution),
		})
	}

	sort.Slice(scores, func(i, j int) bool { return scores[i].Temperature < scores[j].Temperature })
	sort.Slice(recommendations, func(i, j int) bool { return recommendations[i].Temperature < recommendations[j].Temperature })

	return domain.TemperatureReview{
		Summary:         strings.TrimSpace(p.Summary),
		BestAccuracy:    bestAccuracy,
		BestCreativity:  bestCreativity,
		BestDiversity:   bestDiversity,
		Differences:     p.Differences,
		Scores:          scores,
		Recommendations: recommendations,
	}, nil
}

func validScore(score int) bool {
	return score >= 0 && score <= 10
}
