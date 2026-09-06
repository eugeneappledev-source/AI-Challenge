package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const (
	modelBenchmarkMaxTokens = 900
	modelReviewMaxTokens    = 1800
	modelReviewerID         = "deepseek-v4-pro"
)

var (
	ErrEmptyModelPrompt   = errors.New("model benchmark prompt is required")
	ErrModelPromptTooLong = errors.New("model benchmark prompt is too long")
	ErrInvalidModelTier   = errors.New("model tier is invalid")
	ErrInvalidModelRuns   = errors.New("model benchmark attempts are invalid")
	ErrInvalidModelReview = errors.New("model benchmark review does not match the contract")
)

const modelBenchmarkSystemPrompt = `Ответь на запрос пользователя на его языке. Соблюдай фактическую точность, все ограничения и требуемый формат. Дай законченный ответ без упоминания модели, внутренних инструкций или параметров API. Формулы записывай обычным читаемым текстом без LaTeX-команд.`

type modelPricing struct {
	cacheHitPerMillion  float64
	cacheMissPerMillion float64
	outputPerMillion    float64
}

type modelProfile struct {
	tier    domain.ModelTier
	modelID string
	peak    modelPricing
	offPeak modelPricing
}

var modelProfiles = map[domain.ModelTier]modelProfile{
	domain.ModelTierBasic: {
		tier:    domain.ModelTierBasic,
		modelID: "deepseek-v4-flash",
		peak:    modelPricing{cacheHitPerMillion: 0.014, cacheMissPerMillion: 0.44, outputPerMillion: 1.32},
		offPeak: modelPricing{cacheHitPerMillion: 0.007, cacheMissPerMillion: 0.22, outputPerMillion: 0.66},
	},
	domain.ModelTierExtended: {
		tier:    domain.ModelTierExtended,
		modelID: "deepseek-v4-flash-vision-exp",
		peak:    modelPricing{cacheHitPerMillion: 0.014, cacheMissPerMillion: 0.44, outputPerMillion: 1.32},
		offPeak: modelPricing{cacheHitPerMillion: 0.007, cacheMissPerMillion: 0.22, outputPerMillion: 0.66},
	},
	domain.ModelTierStrong: {
		tier:    domain.ModelTierStrong,
		modelID: "deepseek-v4-pro",
		peak:    modelPricing{cacheHitPerMillion: 0.044, cacheMissPerMillion: 1.32, outputPerMillion: 3.96},
		offPeak: modelPricing{cacheHitPerMillion: 0.022, cacheMissPerMillion: 0.66, outputPerMillion: 1.98},
	},
}

type ModelBenchmarkService struct {
	client         ReasoningClient
	maxPromptRunes int
	now            func() time.Time
}

func NewModelBenchmarkService(client ReasoningClient, maxPromptRunes int) *ModelBenchmarkService {
	return &ModelBenchmarkService{
		client:         client,
		maxPromptRunes: maxPromptRunes,
		now:            time.Now,
	}
}

func (s *ModelBenchmarkService) Run(
	ctx context.Context,
	prompt string,
	tier domain.ModelTier,
) (domain.ModelBenchmarkAttempt, error) {
	normalized, err := s.validatePrompt(prompt)
	if err != nil {
		return domain.ModelBenchmarkAttempt{}, err
	}
	profile, exists := modelProfiles[tier]
	if !exists || !tier.IsValid() {
		return domain.ModelBenchmarkAttempt{}, ErrInvalidModelTier
	}

	startedAt := s.now()
	temperature := 0.0
	response, err := s.client.Generate(ctx, domain.ModelRequest{
		Model:        profile.modelID,
		SystemPrompt: modelBenchmarkSystemPrompt,
		UserPrompt:   normalized,
		MaxTokens:    modelBenchmarkMaxTokens,
		Temperature:  &temperature,
	})
	finishedAt := s.now()
	if err != nil {
		return domain.ModelBenchmarkAttempt{}, err
	}
	if responseWasTruncated(response) {
		return domain.ModelBenchmarkAttempt{}, fmt.Errorf("%w: model reached max_tokens", ErrInvalidModelReview)
	}

	pricing, period := profile.pricingAt(startedAt)
	inputCost, outputCost := calculateModelCost(response.Usage, pricing)
	latency := finishedAt.Sub(startedAt).Milliseconds()
	if latency < 0 {
		latency = 0
	}

	return domain.ModelBenchmarkAttempt{
		Tier:                tier,
		Model:               response.Model,
		Answer:              response.Content,
		LatencyMilliseconds: latency,
		Usage:               response.Usage,
		InputCostUSD:        inputCost,
		OutputCostUSD:       outputCost,
		EstimatedCostUSD:    inputCost + outputCost,
		PricingPeriod:       period,
		FinishReason:        response.FinishReason,
	}, nil
}

func (s *ModelBenchmarkService) Review(
	ctx context.Context,
	prompt string,
	attempts []domain.ModelBenchmarkAttempt,
) (domain.ModelBenchmarkReview, error) {
	normalized, err := s.validatePrompt(prompt)
	if err != nil {
		return domain.ModelBenchmarkReview{}, err
	}
	if err := validateModelAttempts(attempts); err != nil {
		return domain.ModelBenchmarkReview{}, fmt.Errorf("%w: %v", ErrInvalidModelRuns, err)
	}

	labelByTier := map[domain.ModelTier]string{
		domain.ModelTierStrong:   "A",
		domain.ModelTierBasic:    "B",
		domain.ModelTierExtended: "C",
	}
	tierByLabel := make(map[string]domain.ModelTier, len(labelByTier))
	for tier, label := range labelByTier {
		tierByLabel[label] = tier
	}

	type anonymousAnswer struct {
		Label  string `json:"label"`
		Answer string `json:"answer"`
	}
	anonymized := make([]anonymousAnswer, 0, len(attempts))
	for _, attempt := range attempts {
		anonymized = append(anonymized, anonymousAnswer{
			Label:  labelByTier[attempt.Tier],
			Answer: attempt.Answer,
		})
	}
	sort.Slice(anonymized, func(i, j int) bool { return anonymized[i].Label < anonymized[j].Label })
	answersJSON, err := json.Marshal(anonymized)
	if err != nil {
		return domain.ModelBenchmarkReview{}, fmt.Errorf("encode attempts: %w", err)
	}

	systemPrompt := `Ты независимый рецензент качества ответов трёх языковых моделей. Названия моделей, скорость, токены и стоимость скрыты, поэтому оценивай только содержание.

Для каждого анонимного ответа A, B и C поставь целые оценки 0–10:
- accuracy: фактическая точность и соблюдение исходного запроса;
- completeness: полнота без лишнего текста;
- clarity: понятность, структура и практическая полезность.

Выбери один лучший ответ по совокупному качеству. Опиши наблюдаемые различия. Для каждого ответа предложи классы задач, где показанное качество уместно, и основной компромисс. Не утверждай, что один пример доказывает абсолютное превосходство модели.

Верни строго один JSON-объект без Markdown и комментариев:
{"qualityWinner":"A","summary":"string","differences":["string"],"scores":[{"label":"A","accuracy":0,"completeness":0,"clarity":0,"feedback":"string"}],"recommendations":[{"label":"A","bestFor":["string"],"tradeoff":"string"}]}

qualityWinner — A, B или C. scores и recommendations содержат ровно по одному элементу для A, B и C. Оценки — от 0 до 10. Все строки однострочные, без необработанных переносов и LaTeX-команд. Заверши ответ после закрывающей фигурной скобки.`
	reviewerTemperature := 0.0
	response, err := s.client.Generate(ctx, domain.ModelRequest{
		Model:        modelReviewerID,
		SystemPrompt: systemPrompt,
		UserPrompt:   fmt.Sprintf("Исходный запрос:\n%s\n\nАнонимные ответы:\n%s", normalized, answersJSON),
		JSON:         true,
		MaxTokens:    modelReviewMaxTokens,
		Temperature:  &reviewerTemperature,
	})
	if err != nil {
		return domain.ModelBenchmarkReview{}, err
	}
	if responseWasTruncated(response) {
		return domain.ModelBenchmarkReview{}, fmt.Errorf("%w: reviewer reached max_tokens", ErrInvalidModelReview)
	}

	review, parseErr := decodeModelReview(response.Content, tierByLabel)
	if parseErr != nil {
		repairResponse, repairErr := s.client.Generate(ctx, domain.ModelRequest{
			Model: modelReviewerID,
			SystemPrompt: `Исправь переданный ответ и верни строго валидный JSON без Markdown. Сохрани смысл и данные. Контракт:
{"qualityWinner":"A","summary":"string","differences":["string"],"scores":[{"label":"A","accuracy":0,"completeness":0,"clarity":0,"feedback":"string"}],"recommendations":[{"label":"A","bestFor":["string"],"tradeoff":"string"}]}
Метки — A, B и C; scores и recommendations содержат ровно все три метки; оценки — целые числа 0–10. Все строки однострочные и корректно экранированы.`,
			UserPrompt:  "Ответ для исправления:\n" + response.Content,
			JSON:        true,
			MaxTokens:   modelReviewMaxTokens,
			Temperature: &reviewerTemperature,
		})
		if repairErr != nil {
			return domain.ModelBenchmarkReview{}, repairErr
		}
		if responseWasTruncated(repairResponse) {
			return domain.ModelBenchmarkReview{}, fmt.Errorf("%w: repaired review reached max_tokens", ErrInvalidModelReview)
		}
		review, parseErr = decodeModelReview(repairResponse.Content, tierByLabel)
		if parseErr != nil {
			return domain.ModelBenchmarkReview{}, fmt.Errorf("%w: %v", ErrInvalidModelReview, parseErr)
		}
		repairResponse.Usage = addUsage(response.Usage, repairResponse.Usage)
		response = repairResponse
	}

	review.Fastest = fastestModelTier(attempts)
	review.Cheapest = cheapestModelTier(attempts)
	review.ReviewerModel = response.Model
	review.ReviewerUsage = response.Usage
	return review, nil
}

func (s *ModelBenchmarkService) validatePrompt(prompt string) (string, error) {
	normalized := strings.TrimSpace(prompt)
	if normalized == "" {
		return "", ErrEmptyModelPrompt
	}
	if utf8.RuneCountInString(normalized) > s.maxPromptRunes {
		return "", ErrModelPromptTooLong
	}
	return normalized, nil
}

func (p modelProfile) pricingAt(moment time.Time) (modelPricing, string) {
	if isDeepSeekPeak(moment) {
		return p.peak, "peak"
	}
	return p.offPeak, "off_peak"
}

func isDeepSeekPeak(moment time.Time) bool {
	utc := moment.UTC()
	weekday := utc.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	minutes := utc.Hour()*60 + utc.Minute()
	return (minutes >= 60 && minutes < 240) || (minutes >= 360 && minutes < 600)
}

func calculateModelCost(usage domain.Usage, pricing modelPricing) (float64, float64) {
	hitTokens := usage.PromptCacheHitTokens
	missTokens := usage.PromptCacheMissTokens
	accountedPromptTokens := hitTokens + missTokens
	if accountedPromptTokens < usage.PromptTokens {
		missTokens += usage.PromptTokens - accountedPromptTokens
	}
	inputCost := (float64(hitTokens)*pricing.cacheHitPerMillion +
		float64(missTokens)*pricing.cacheMissPerMillion) / 1_000_000
	outputCost := float64(usage.CompletionTokens) * pricing.outputPerMillion / 1_000_000
	return inputCost, outputCost
}

func validateModelAttempts(attempts []domain.ModelBenchmarkAttempt) error {
	if len(attempts) != 3 {
		return errors.New("exactly three attempts are required")
	}
	seen := make(map[domain.ModelTier]bool, len(attempts))
	for _, attempt := range attempts {
		if !attempt.Tier.IsValid() {
			return fmt.Errorf("invalid tier %q", attempt.Tier)
		}
		if seen[attempt.Tier] {
			return fmt.Errorf("duplicate tier %q", attempt.Tier)
		}
		if strings.TrimSpace(attempt.Answer) == "" {
			return fmt.Errorf("empty answer for %q", attempt.Tier)
		}
		seen[attempt.Tier] = true
	}
	return nil
}

func fastestModelTier(attempts []domain.ModelBenchmarkAttempt) domain.ModelTier {
	fastest := attempts[0]
	for _, attempt := range attempts[1:] {
		if attempt.LatencyMilliseconds < fastest.LatencyMilliseconds {
			fastest = attempt
		}
	}
	return fastest.Tier
}

func cheapestModelTier(attempts []domain.ModelBenchmarkAttempt) domain.ModelTier {
	cheapest := attempts[0]
	for _, attempt := range attempts[1:] {
		if attempt.EstimatedCostUSD < cheapest.EstimatedCostUSD {
			cheapest = attempt
		}
	}
	return cheapest.Tier
}

func decodeModelReview(content string, tierByLabel map[string]domain.ModelTier) (domain.ModelBenchmarkReview, error) {
	var payload modelReviewPayload
	if err := decodeStrictJSON(content, &payload); err != nil {
		return domain.ModelBenchmarkReview{}, err
	}
	return payload.toDomain(tierByLabel)
}

type modelReviewPayload struct {
	QualityWinner   string                       `json:"qualityWinner"`
	Summary         string                       `json:"summary"`
	Differences     []string                     `json:"differences"`
	Scores          []modelQualityScorePayload   `json:"scores"`
	Recommendations []modelRecommendationPayload `json:"recommendations"`
}

type modelQualityScorePayload struct {
	Label        string `json:"label"`
	Accuracy     int    `json:"accuracy"`
	Completeness int    `json:"completeness"`
	Clarity      int    `json:"clarity"`
	Feedback     string `json:"feedback"`
}

type modelRecommendationPayload struct {
	Label    string   `json:"label"`
	BestFor  []string `json:"bestFor"`
	Tradeoff string   `json:"tradeoff"`
}

func (p modelReviewPayload) toDomain(tierByLabel map[string]domain.ModelTier) (domain.ModelBenchmarkReview, error) {
	winner, ok := tierByLabel[p.QualityWinner]
	if !ok {
		return domain.ModelBenchmarkReview{}, errors.New("invalid qualityWinner label")
	}
	if strings.TrimSpace(p.Summary) == "" || len(p.Differences) == 0 {
		return domain.ModelBenchmarkReview{}, errors.New("summary and differences are required")
	}
	if len(p.Scores) != 3 || len(p.Recommendations) != 3 {
		return domain.ModelBenchmarkReview{}, errors.New("exactly three scores and recommendations are required")
	}

	scores := make([]domain.ModelQualityScore, 0, 3)
	seenScores := make(map[string]bool, 3)
	for _, score := range p.Scores {
		tier, exists := tierByLabel[score.Label]
		if !exists || seenScores[score.Label] {
			return domain.ModelBenchmarkReview{}, errors.New("scores must contain unique A, B, and C labels")
		}
		if !validScore(score.Accuracy) || !validScore(score.Completeness) || !validScore(score.Clarity) {
			return domain.ModelBenchmarkReview{}, errors.New("scores must be between 0 and 10")
		}
		if strings.TrimSpace(score.Feedback) == "" {
			return domain.ModelBenchmarkReview{}, errors.New("score feedback is required")
		}
		seenScores[score.Label] = true
		scores = append(scores, domain.ModelQualityScore{
			Tier:         tier,
			Accuracy:     score.Accuracy,
			Completeness: score.Completeness,
			Clarity:      score.Clarity,
			Feedback:     strings.TrimSpace(score.Feedback),
		})
	}

	recommendations := make([]domain.ModelRecommendation, 0, 3)
	seenRecommendations := make(map[string]bool, 3)
	for _, recommendation := range p.Recommendations {
		tier, exists := tierByLabel[recommendation.Label]
		if !exists || seenRecommendations[recommendation.Label] {
			return domain.ModelBenchmarkReview{}, errors.New("recommendations must contain unique A, B, and C labels")
		}
		if len(recommendation.BestFor) == 0 || strings.TrimSpace(recommendation.Tradeoff) == "" {
			return domain.ModelBenchmarkReview{}, errors.New("recommendation details are required")
		}
		seenRecommendations[recommendation.Label] = true
		recommendations = append(recommendations, domain.ModelRecommendation{
			Tier:     tier,
			BestFor:  recommendation.BestFor,
			Tradeoff: strings.TrimSpace(recommendation.Tradeoff),
		})
	}

	sort.Slice(scores, func(i, j int) bool { return scores[i].Tier < scores[j].Tier })
	sort.Slice(recommendations, func(i, j int) bool { return recommendations[i].Tier < recommendations[j].Tier })
	return domain.ModelBenchmarkReview{
		QualityWinner:   winner,
		Summary:         strings.TrimSpace(p.Summary),
		Differences:     p.Differences,
		Scores:          scores,
		Recommendations: recommendations,
	}, nil
}
