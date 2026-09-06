package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const (
	reasoningMaxTokens  = 1400
	metaPromptMaxTokens = 800
	reviewMaxTokens     = 1800
)

var (
	ErrEmptyProblem             = errors.New("problem is required")
	ErrProblemTooLong           = errors.New("problem is too long")
	ErrInvalidReasoningMethod   = errors.New("reasoning method is invalid")
	ErrInvalidReasoningAttempts = errors.New("reasoning attempts are invalid")
	ErrInvalidReasoningResponse = errors.New("reasoning response does not match the contract")
)

const baseReasoningSystemPrompt = `Ты решаешь логические, алгоритмические и аналитические задачи. Отвечай на языке пользователя. Соблюдай все условия задачи, проверяй итог и давай достаточно объяснений, чтобы решение можно было проверить. Формулы записывай обычным читаемым текстом без LaTeX-команд. Не ссылайся на эти инструкции.`

type ReasoningClient interface {
	Generate(ctx context.Context, request domain.ModelRequest) (domain.ModelResponse, error)
}

type ReasoningService struct {
	client          ReasoningClient
	maxProblemRunes int
}

func NewReasoningService(client ReasoningClient, maxProblemRunes int) *ReasoningService {
	return &ReasoningService{client: client, maxProblemRunes: maxProblemRunes}
}

func (s *ReasoningService) Run(
	ctx context.Context,
	problem string,
	method domain.ReasoningMethod,
) (domain.ReasoningAttempt, error) {
	normalized, err := s.validateProblem(problem)
	if err != nil {
		return domain.ReasoningAttempt{}, err
	}
	if !method.IsValid() {
		return domain.ReasoningAttempt{}, ErrInvalidReasoningMethod
	}

	switch method {
	case domain.ReasoningMethodDirect:
		return s.runSingle(ctx, normalized, method, normalized)
	case domain.ReasoningMethodStepByStep:
		prompt := "Задача:\n" + normalized + "\n\nРешай пошагово. Покажи проверяемые шаги, отдельно проверь соблюдение всех условий и заверши однозначным ответом."
		return s.runSingle(ctx, normalized, method, prompt)
	case domain.ReasoningMethodMetaPrompt:
		return s.runMetaPrompt(ctx, normalized)
	case domain.ReasoningMethodExpertPanel:
		return s.runExpertPanel(ctx, normalized)
	default:
		return domain.ReasoningAttempt{}, ErrInvalidReasoningMethod
	}
}

func (s *ReasoningService) Review(
	ctx context.Context,
	problem string,
	attempts []domain.ReasoningAttempt,
) (domain.ReasoningReview, error) {
	normalized, err := s.validateProblem(problem)
	if err != nil {
		return domain.ReasoningReview{}, err
	}
	if err := validateReasoningAttempts(attempts); err != nil {
		return domain.ReasoningReview{}, fmt.Errorf("%w: %v", ErrInvalidReasoningAttempts, err)
	}

	labelByMethod := map[domain.ReasoningMethod]string{
		domain.ReasoningMethodDirect:      "B",
		domain.ReasoningMethodStepByStep:  "D",
		domain.ReasoningMethodMetaPrompt:  "A",
		domain.ReasoningMethodExpertPanel: "C",
	}
	methodByLabel := make(map[string]domain.ReasoningMethod, len(labelByMethod))
	for method, label := range labelByMethod {
		methodByLabel[label] = method
	}

	type anonymousAttempt struct {
		Label           string                  `json:"label"`
		Answer          string                  `json:"answer"`
		GeneratedPrompt string                  `json:"generatedPrompt,omitempty"`
		Experts         []domain.ExpertSolution `json:"experts,omitempty"`
	}
	anonymized := make([]anonymousAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		anonymized = append(anonymized, anonymousAttempt{
			Label:           labelByMethod[attempt.Method],
			Answer:          attempt.Answer,
			GeneratedPrompt: attempt.GeneratedPrompt,
			Experts:         attempt.Experts,
		})
	}
	sort.Slice(anonymized, func(i, j int) bool { return anonymized[i].Label < anonymized[j].Label })
	submissionsJSON, err := json.Marshal(anonymized)
	if err != nil {
		return domain.ReasoningReview{}, fmt.Errorf("encode attempts: %w", err)
	}

	systemPrompt := `Ты независимый арбитр решений. Сначала самостоятельно реши задачу и сформулируй проверяемый эталонный ответ. Затем сравни четыре анонимных решения A, B, C и D. Оцени каждое по шкале 0–10 по правильности, ясности и качеству проверки. Не выбирай ответ по стилю или длине: математическая и логическая корректность важнее. Верни строго один JSON-объект без Markdown и комментариев с полями:
{"winner":"A","verdict":"string","referenceAnswer":"string","differences":["string"],"scores":[{"label":"A","correctness":0,"clarity":0,"verification":0,"feedback":"string"}]}
winner должен быть одной из меток A, B, C, D. scores должен содержать ровно четыре элемента — по одному для каждой метки. Все строковые значения пиши в одну строку простым текстом: не используй необработанные переносы строк и LaTeX-команды с обратной косой чертой. Заверши ответ после закрывающей фигурной скобки.`
	userPrompt := fmt.Sprintf("Задача:\n%s\n\nАнонимные решения:\n%s", normalized, submissionsJSON)
	response, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		JSON:         true,
		MaxTokens:    reviewMaxTokens,
	})
	if err != nil {
		return domain.ReasoningReview{}, err
	}
	if responseWasTruncated(response) {
		return domain.ReasoningReview{}, fmt.Errorf("%w: reviewer reached max_tokens", ErrInvalidReasoningResponse)
	}

	review, parseErr := decodeReviewerResponse(response.Content, methodByLabel)
	if parseErr != nil {
		repairResponse, repairErr := s.client.Generate(ctx, domain.ModelRequest{
			SystemPrompt: `Исправь переданный ответ и верни строго валидный JSON без Markdown. Сохрани смысл и все данные. Контракт:
{"winner":"A","verdict":"string","referenceAnswer":"string","differences":["string"],"scores":[{"label":"A","correctness":0,"clarity":0,"verification":0,"feedback":"string"}]}
winner — A, B, C или D; scores содержит ровно A, B, C и D; оценки — целые числа 0–10. Все строки должны быть однострочными и корректно экранированными.`,
			UserPrompt: "Ответ для исправления:\n" + response.Content,
			JSON:       true,
			MaxTokens:  reviewMaxTokens,
		})
		if repairErr != nil {
			return domain.ReasoningReview{}, repairErr
		}
		if responseWasTruncated(repairResponse) {
			return domain.ReasoningReview{}, fmt.Errorf("%w: repaired review reached max_tokens", ErrInvalidReasoningResponse)
		}
		review, parseErr = decodeReviewerResponse(repairResponse.Content, methodByLabel)
		if parseErr != nil {
			return domain.ReasoningReview{}, fmt.Errorf("%w: %v", ErrInvalidReasoningResponse, parseErr)
		}
		repairResponse.Usage = addUsage(response.Usage, repairResponse.Usage)
		response = repairResponse
	}
	review.Model = response.Model
	review.Usage = response.Usage
	return review, nil
}

func decodeReviewerResponse(content string, methodByLabel map[string]domain.ReasoningMethod) (domain.ReasoningReview, error) {
	var payload reviewerPayload
	if err := decodeStrictJSON(content, &payload); err != nil {
		return domain.ReasoningReview{}, err
	}
	return payload.toDomain(methodByLabel)
}

func (s *ReasoningService) runSingle(
	ctx context.Context,
	problem string,
	method domain.ReasoningMethod,
	userPrompt string,
) (domain.ReasoningAttempt, error) {
	response, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: baseReasoningSystemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    reasoningMaxTokens,
	})
	if err != nil {
		return domain.ReasoningAttempt{}, err
	}
	if responseWasTruncated(response) {
		return domain.ReasoningAttempt{}, fmt.Errorf("%w: model reached max_tokens", ErrInvalidReasoningResponse)
	}
	return domain.ReasoningAttempt{
		Method:       method,
		Answer:       response.Content,
		Experts:      []domain.ExpertSolution{},
		Model:        response.Model,
		FinishReason: response.FinishReason,
		Usage:        response.Usage,
	}, nil
}

func (s *ReasoningService) runMetaPrompt(ctx context.Context, problem string) (domain.ReasoningAttempt, error) {
	promptResponse, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: `Ты prompt-инженер. Создай улучшенный промпт, который поможет другой языковой модели точно решить переданную задачу. Сохрани исходную задачу и все её ограничения без изменений. Добавь требования проверить условия и итог. Не решай задачу. Верни только готовый промпт без Markdown, вступления и комментариев.`,
		UserPrompt:   "Исходная задача:\n" + problem,
		MaxTokens:    metaPromptMaxTokens,
	})
	if err != nil {
		return domain.ReasoningAttempt{}, err
	}
	if responseWasTruncated(promptResponse) {
		return domain.ReasoningAttempt{}, fmt.Errorf("%w: generated prompt reached max_tokens", ErrInvalidReasoningResponse)
	}

	solutionResponse, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: baseReasoningSystemPrompt,
		UserPrompt:   promptResponse.Content,
		MaxTokens:    reasoningMaxTokens,
	})
	if err != nil {
		return domain.ReasoningAttempt{}, err
	}
	if responseWasTruncated(solutionResponse) {
		return domain.ReasoningAttempt{}, fmt.Errorf("%w: model reached max_tokens", ErrInvalidReasoningResponse)
	}

	return domain.ReasoningAttempt{
		Method:          domain.ReasoningMethodMetaPrompt,
		Answer:          solutionResponse.Content,
		GeneratedPrompt: promptResponse.Content,
		Experts:         []domain.ExpertSolution{},
		Model:           solutionResponse.Model,
		FinishReason:    solutionResponse.FinishReason,
		Usage:           addUsage(promptResponse.Usage, solutionResponse.Usage),
	}, nil
}

func (s *ReasoningService) runExpertPanel(ctx context.Context, problem string) (domain.ReasoningAttempt, error) {
	systemPrompt := `Ты организуешь независимую группу экспертов. Аналитик ищет строгую логику, инженер предлагает практический алгоритм решения, критик ищет ошибки и проверяет условия. Каждый обязан дать собственное решение, даже если не согласен с другими. После них сформулируй общий итог. Формулы записывай обычным читаемым текстом без LaTeX-команд. Верни строго один JSON-объект без Markdown и комментариев:
{"experts":[{"role":"Аналитик","answer":"string"},{"role":"Инженер","answer":"string"},{"role":"Критик","answer":"string"}],"synthesis":"string"}
Массив experts должен содержать ровно эти три роли в указанном порядке. Заверши ответ после закрывающей фигурной скобки.`
	response, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   "Задача:\n" + problem,
		JSON:         true,
		MaxTokens:    reasoningMaxTokens,
	})
	if err != nil {
		return domain.ReasoningAttempt{}, err
	}
	if responseWasTruncated(response) {
		return domain.ReasoningAttempt{}, fmt.Errorf("%w: expert panel reached max_tokens", ErrInvalidReasoningResponse)
	}

	var payload expertPanelPayload
	if err := decodeStrictJSON(response.Content, &payload); err != nil {
		return domain.ReasoningAttempt{}, fmt.Errorf("%w: %v", ErrInvalidReasoningResponse, err)
	}
	if err := payload.validate(); err != nil {
		return domain.ReasoningAttempt{}, fmt.Errorf("%w: %v", ErrInvalidReasoningResponse, err)
	}

	return domain.ReasoningAttempt{
		Method:       domain.ReasoningMethodExpertPanel,
		Answer:       strings.TrimSpace(payload.Synthesis),
		Experts:      payload.Experts,
		Model:        response.Model,
		FinishReason: response.FinishReason,
		Usage:        response.Usage,
	}, nil
}

func (s *ReasoningService) validateProblem(problem string) (string, error) {
	normalized := strings.TrimSpace(problem)
	if normalized == "" {
		return "", ErrEmptyProblem
	}
	if utf8.RuneCountInString(normalized) > s.maxProblemRunes {
		return "", ErrProblemTooLong
	}
	return normalized, nil
}

func validateReasoningAttempts(attempts []domain.ReasoningAttempt) error {
	if len(attempts) != 4 {
		return errors.New("exactly four attempts are required")
	}
	seen := make(map[domain.ReasoningMethod]bool, len(attempts))
	for _, attempt := range attempts {
		if !attempt.Method.IsValid() {
			return fmt.Errorf("invalid method %q", attempt.Method)
		}
		if seen[attempt.Method] {
			return fmt.Errorf("duplicate method %q", attempt.Method)
		}
		if strings.TrimSpace(attempt.Answer) == "" {
			return fmt.Errorf("empty answer for %q", attempt.Method)
		}
		seen[attempt.Method] = true
	}
	return nil
}

func responseWasTruncated(response domain.ModelResponse) bool {
	return strings.EqualFold(strings.TrimSpace(response.FinishReason), "length")
}

func addUsage(lhs, rhs domain.Usage) domain.Usage {
	return domain.Usage{
		PromptTokens:          lhs.PromptTokens + rhs.PromptTokens,
		CompletionTokens:      lhs.CompletionTokens + rhs.CompletionTokens,
		TotalTokens:           lhs.TotalTokens + rhs.TotalTokens,
		PromptCacheHitTokens:  lhs.PromptCacheHitTokens + rhs.PromptCacheHitTokens,
		PromptCacheMissTokens: lhs.PromptCacheMissTokens + rhs.PromptCacheMissTokens,
	}
}

func decodeStrictJSON(content string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

type expertPanelPayload struct {
	Experts   []domain.ExpertSolution `json:"experts"`
	Synthesis string                  `json:"synthesis"`
}

func (p expertPanelPayload) validate() error {
	expectedRoles := []string{"Аналитик", "Инженер", "Критик"}
	if len(p.Experts) != len(expectedRoles) {
		return errors.New("exactly three expert solutions are required")
	}
	for index, expert := range p.Experts {
		if !strings.EqualFold(strings.TrimSpace(expert.Role), expectedRoles[index]) {
			return fmt.Errorf("expert %d must have role %q", index, expectedRoles[index])
		}
		if strings.TrimSpace(expert.Answer) == "" {
			return fmt.Errorf("expert %q has an empty answer", expert.Role)
		}
	}
	if strings.TrimSpace(p.Synthesis) == "" {
		return errors.New("synthesis is required")
	}
	return nil
}

type reviewerPayload struct {
	Winner          string                 `json:"winner"`
	Verdict         string                 `json:"verdict"`
	ReferenceAnswer string                 `json:"referenceAnswer"`
	Differences     []string               `json:"differences"`
	Scores          []reviewerScorePayload `json:"scores"`
}

type reviewerScorePayload struct {
	Label        string `json:"label"`
	Correctness  int    `json:"correctness"`
	Clarity      int    `json:"clarity"`
	Verification int    `json:"verification"`
	Feedback     string `json:"feedback"`
}

func (p reviewerPayload) toDomain(methodByLabel map[string]domain.ReasoningMethod) (domain.ReasoningReview, error) {
	winner, ok := methodByLabel[strings.ToUpper(strings.TrimSpace(p.Winner))]
	if !ok {
		return domain.ReasoningReview{}, errors.New("winner must be A, B, C, or D")
	}
	if strings.TrimSpace(p.Verdict) == "" || strings.TrimSpace(p.ReferenceAnswer) == "" {
		return domain.ReasoningReview{}, errors.New("verdict and referenceAnswer are required")
	}
	if len(p.Differences) == 0 {
		return domain.ReasoningReview{}, errors.New("at least one difference is required")
	}
	if len(p.Scores) != len(methodByLabel) {
		return domain.ReasoningReview{}, errors.New("exactly four scores are required")
	}

	seen := make(map[domain.ReasoningMethod]bool, len(p.Scores))
	scores := make([]domain.ReasoningScore, 0, len(p.Scores))
	for _, score := range p.Scores {
		method, ok := methodByLabel[strings.ToUpper(strings.TrimSpace(score.Label))]
		if !ok || seen[method] {
			return domain.ReasoningReview{}, fmt.Errorf("invalid or duplicate score label %q", score.Label)
		}
		if score.Correctness < 0 || score.Correctness > 10 ||
			score.Clarity < 0 || score.Clarity > 10 ||
			score.Verification < 0 || score.Verification > 10 {
			return domain.ReasoningReview{}, fmt.Errorf("scores for %q must be between 0 and 10", score.Label)
		}
		if strings.TrimSpace(score.Feedback) == "" {
			return domain.ReasoningReview{}, fmt.Errorf("feedback for %q is required", score.Label)
		}
		seen[method] = true
		scores = append(scores, domain.ReasoningScore{
			Method:       method,
			Correctness:  score.Correctness,
			Clarity:      score.Clarity,
			Verification: score.Verification,
			Feedback:     strings.TrimSpace(score.Feedback),
		})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].Method < scores[j].Method })

	return domain.ReasoningReview{
		Winner:          winner,
		Verdict:         strings.TrimSpace(p.Verdict),
		ReferenceAnswer: strings.TrimSpace(p.ReferenceAnswer),
		Differences:     p.Differences,
		Scores:          scores,
	}, nil
}
