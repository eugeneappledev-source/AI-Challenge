package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var (
	ErrEmptyRAGQuestion   = errors.New("question is required")
	ErrRAGQuestionTooLong = errors.New("question is too long")
	ErrInvalidRAGSession  = errors.New("session id is invalid")
)

type RAGStore interface {
	LoadKnowledgeChunks(context.Context, domain.ChunkStrategy) ([]domain.KnowledgeChunk, error)
	AppendRAGChatMessage(context.Context, string, domain.RAGChatMessage) (domain.RAGChatMessage, error)
	LoadRAGChat(context.Context, string) ([]domain.RAGChatMessage, error)
	LoadRAGTaskState(context.Context, string) (domain.RAGTaskState, bool, error)
	SaveRAGTaskState(context.Context, domain.RAGTaskState) error
	ClearRAGChat(context.Context, string) error
}

type RAGIndexer interface {
	Build(context.Context) (domain.KnowledgeIndexStatus, error)
}

type RAGService struct {
	store    RAGStore
	indexer  RAGIndexer
	client   ReasoningClient
	maxRunes int
}

func NewRAGService(store RAGStore, indexer RAGIndexer, client ReasoningClient, maxRunes int) *RAGService {
	return &RAGService{store: store, indexer: indexer, client: client, maxRunes: maxRunes}
}

func (s *RAGService) ControlQuestions() []domain.ControlQuestion {
	return []domain.ControlQuestion{
		{ID: "q01", Question: "Как проект получает HTTPS на VPS?", ExpectedSources: []string{"deploy/Caddyfile", "deploy/compose.yaml"}},
		{ID: "q02", Question: "Какие два MCP-сервера оркестрирует День 20?", ExpectedSources: []string{"challenges/day-20/README.md", "backend/internal/application/business_network.go"}},
		{ID: "q03", Question: "Где сохраняется история автоматических дайджестов?", ExpectedSources: []string{"backend/internal/infrastructure/sqlite/radar_store.go"}},
		{ID: "q04", Question: "Как запускается агент Дня 18 по расписанию?", ExpectedSources: []string{"challenges/day-18/README.md", "deploy/compose.yaml"}},
		{ID: "q05", Question: "Какие режимы ответа сравниваются во втором задании?", ExpectedSources: []string{"challenges/day-02/README.md"}},
		{ID: "q06", Question: "Какие фазы есть у состояния задачи агента?", ExpectedSources: []string{"backend/internal/domain/task_state.go", "challenges/day-13/README.md"}},
		{ID: "q07", Question: "Какие три стратегии контекста реализованы в Дне 10?", ExpectedSources: []string{"challenges/day-10/README.md"}},
		{ID: "q08", Question: "Какие инструменты входят в Research MCP server?", ExpectedSources: []string{"backend/internal/infrastructure/mcpserver/research.go", "challenges/day-19/README.md"}},
		{ID: "q09", Question: "Как web-приложение переключает лаборатории разных дней?", ExpectedSources: []string{"web/src/App.tsx"}},
		{ID: "q10", Question: "Почему API-ключ модели не попадает в браузер?", ExpectedSources: []string{"deploy/Caddyfile", "README.md"}},
	}
}

func (s *RAGService) Compare(ctx context.Context, question string) (domain.RAGComparison, error) {
	normalized, err := s.validateQuestion(question)
	if err != nil {
		return domain.RAGComparison{}, err
	}
	retrieved, err := s.Retrieve(ctx, normalized, 5)
	if err != nil {
		return domain.RAGComparison{}, err
	}
	without, err := s.generate(ctx, `Ты отвечаешь на вопрос пользователя без доступа к базе знаний проекта. Не выдумывай детали: если точных данных нет, честно скажи об этом. Отвечай кратко на языке пользователя.`, normalized)
	if err != nil {
		return domain.RAGComparison{}, fmt.Errorf("answer without RAG: %w", err)
	}
	contextBlock := formatRetrievedContext(retrieved)
	with, err := s.generate(ctx, `Ты отвечаешь только по переданному контексту из репозитория AI Challenge. Сформулируй точный законченный ответ на языке пользователя. Не используй внешние знания и не придумывай отсутствующие факты. В конце назови пути использованных файлов.`, "Вопрос:\n"+normalized+"\n\nКонтекст:\n"+contextBlock)
	if err != nil {
		return domain.RAGComparison{}, fmt.Errorf("answer with RAG: %w", err)
	}
	return domain.RAGComparison{Question: normalized, WithoutRAG: without, WithRAG: with, Retrieved: retrieved}, nil
}

func (s *RAGService) Retrieve(ctx context.Context, question string, topK int) ([]domain.RetrievedChunk, error) {
	chunks, err := s.loadChunks(ctx)
	if err != nil {
		return nil, err
	}
	queryVector := embedText(question)
	results := make([]domain.RetrievedChunk, 0, len(chunks))
	for _, chunk := range chunks {
		results = append(results, domain.RetrievedChunk{Chunk: chunk, SimilarityScore: cosine(queryVector, chunk.Vector)})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].SimilarityScore > results[j].SimilarityScore })
	if topK < 1 {
		topK = 1
	}
	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

func (s *RAGService) CompareRetrieval(ctx context.Context, question string, topK int, threshold float64) (domain.RetrievalComparison, error) {
	started := time.Now()
	normalized, err := s.validateQuestion(question)
	if err != nil {
		return domain.RetrievalComparison{}, err
	}
	if topK < 1 {
		topK = 1
	}
	if topK > 10 {
		topK = 10
	}
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}
	baseline, err := s.Retrieve(ctx, normalized, topK)
	if err != nil {
		return domain.RetrievalComparison{}, err
	}
	temperature := 0.0
	rewrite, err := s.client.Generate(ctx, domain.ModelRequest{SystemPrompt: `Переформулируй вопрос в один точный поисковый запрос для локального индекса репозитория. Добавь вероятные технические термины и имена файлов, но не отвечай на вопрос. Верни только запрос одной строкой без кавычек.`, UserPrompt: normalized, Temperature: &temperature, MaxTokens: 120})
	if err != nil {
		return domain.RetrievalComparison{}, fmt.Errorf("rewrite retrieval query: %w", err)
	}
	rewritten := strings.TrimSpace(rewrite.Content)
	if rewritten == "" {
		rewritten = normalized
	}
	candidateCount := topK * 4
	if candidateCount < 12 {
		candidateCount = 12
	}
	candidates, err := s.Retrieve(ctx, rewritten, candidateCount)
	if err != nil {
		return domain.RetrievalComparison{}, err
	}
	queryTokens := tokenSet(normalized + " " + rewritten)
	for index := range candidates {
		chunk := &candidates[index]
		chunk.LexicalScore = lexicalOverlap(queryTokens, tokenSet(chunk.Chunk.Source+" "+chunk.Chunk.Section+" "+chunk.Chunk.Content))
		metadata := 0.0
		if lexicalOverlap(queryTokens, tokenSet(chunk.Chunk.Source+" "+chunk.Chunk.Section)) > 0 {
			metadata = 1
		}
		chunk.RerankScore = .72*chunk.SimilarityScore + .23*chunk.LexicalScore + .05*metadata
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].RerankScore > candidates[j].RerankScore })
	improved := make([]domain.RetrievedChunk, 0, topK)
	for _, item := range candidates {
		if item.RerankScore >= threshold && len(improved) < topK {
			improved = append(improved, item)
		}
	}
	return domain.RetrievalComparison{Question: normalized, RewrittenQuery: rewritten, TopK: topK, Threshold: threshold, Baseline: baseline, Candidates: candidates, Improved: improved, Dropped: len(candidates) - len(improved), RewriteModel: rewrite.Model, RewriteUsage: rewrite.Usage, DurationMS: time.Since(started).Milliseconds()}, nil
}

func (s *RAGService) AnswerGrounded(ctx context.Context, question string, threshold float64) (domain.GroundedAnswer, error) {
	started := time.Now()
	normalized, err := s.validateQuestion(question)
	if err != nil {
		return domain.GroundedAnswer{}, err
	}
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}
	retrieval, err := s.CompareRetrieval(ctx, normalized, 5, threshold)
	if err != nil {
		return domain.GroundedAnswer{}, err
	}
	if len(retrieval.Improved) == 0 {
		return domain.GroundedAnswer{Status: "insufficient_context", Question: normalized, Answer: "В базе знаний недостаточно подтверждённого контекста для надёжного ответа.", Threshold: threshold, Citations: []domain.EvidenceCitation{}, DurationMS: time.Since(started).Milliseconds()}, nil
	}
	citations := make([]domain.EvidenceCitation, 0, 3)
	for _, item := range retrieval.Improved {
		if len(citations) == 3 {
			break
		}
		citations = append(citations, citationFrom(item))
	}
	contextBlock := formatRetrievedContext(retrieval.Improved)
	temperature := 0.1
	response, err := s.client.Generate(ctx, domain.ModelRequest{SystemPrompt: `Ты evidence-first ассистент по репозиторию AI Challenge. Ответь только по переданным фрагментам. Не выдумывай факты, пути или цитаты. Делай ссылки вида [1], [2] на номера фрагментов. Если фрагменты противоречат друг другу, укажи это.`, UserPrompt: "Вопрос:\n" + normalized + "\n\nПроверенный контекст:\n" + contextBlock, Temperature: &temperature, MaxTokens: 900})
	if err != nil {
		return domain.GroundedAnswer{}, fmt.Errorf("generate grounded answer: %w", err)
	}
	return domain.GroundedAnswer{Status: "answered", Question: normalized, Answer: strings.TrimSpace(response.Content), Confidence: retrieval.Improved[0].RerankScore, Threshold: threshold, Citations: citations, Model: response.Model, Usage: response.Usage, DurationMS: time.Since(started).Milliseconds()}, nil
}

func (s *RAGService) EvaluateEvidence(ctx context.Context, threshold float64) (domain.EvidenceReport, error) {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}
	questions := s.ControlQuestions()
	report := domain.EvidenceReport{Threshold: threshold, Checks: make([]domain.EvidenceCheck, 0, len(questions)), Total: len(questions)}
	for _, question := range questions {
		results, err := s.Retrieve(ctx, question.Question, 5)
		if err != nil {
			return report, err
		}
		check := domain.EvidenceCheck{ID: question.ID, Question: question.Question, ExpectedSources: question.ExpectedSources}
		if len(results) > 0 {
			item := results[0]
			citation := citationFrom(item)
			check.TopSource = item.Chunk.Source
			check.Score = item.SimilarityScore
			check.AboveThreshold = item.SimilarityScore >= threshold
			check.QuoteValid = citation.QuoteValid
			check.Citation = &citation
			for _, candidate := range results {
				for _, expected := range question.ExpectedSources {
					if candidate.Chunk.Source == expected {
						check.ExpectedSourceMatched = true
						break
					}
				}
				if check.ExpectedSourceMatched {
					break
				}
			}
		}
		if check.ExpectedSourceMatched && check.AboveThreshold && check.QuoteValid {
			report.Passed++
		}
		report.Checks = append(report.Checks, check)
	}
	return report, nil
}

func (s *RAGService) Chat(ctx context.Context, sessionID, input string) (domain.RAGChatExchange, error) {
	sessionID = strings.TrimSpace(sessionID)
	if !validRAGSession(sessionID) {
		return domain.RAGChatExchange{}, ErrInvalidRAGSession
	}
	normalized, err := s.validateQuestion(input)
	if err != nil {
		return domain.RAGChatExchange{}, err
	}
	history, err := s.store.LoadRAGChat(ctx, sessionID)
	if err != nil {
		return domain.RAGChatExchange{}, err
	}
	task, _, err := s.store.LoadRAGTaskState(ctx, sessionID)
	if err != nil {
		return domain.RAGChatExchange{}, err
	}
	task = updateRAGTaskState(task, sessionID, normalized, time.Now().UTC())
	if err := s.store.SaveRAGTaskState(ctx, task); err != nil {
		return domain.RAGChatExchange{}, err
	}
	query := task.Goal + "\n" + strings.Join(task.Constraints, "\n") + "\n" + normalized
	retrieval, err := s.CompareRetrieval(ctx, query, 5, .06)
	if err != nil {
		return domain.RAGChatExchange{}, err
	}
	user := domain.RAGChatMessage{Role: "user", Content: normalized, Citations: []domain.EvidenceCitation{}, CreatedAt: time.Now().UTC()}
	assistant := domain.RAGChatMessage{Role: "assistant", Citations: []domain.EvidenceCitation{}, CreatedAt: time.Now().UTC()}
	if len(retrieval.Improved) == 0 {
		assistant.Content = "В базе знаний недостаточно подтверждённого контекста, чтобы продолжить задачу без догадок."
	} else {
		for _, item := range retrieval.Improved {
			if len(assistant.Citations) == 3 {
				break
			}
			assistant.Citations = append(assistant.Citations, citationFrom(item))
		}
		recent := history
		if len(recent) > 8 {
			recent = recent[len(recent)-8:]
		}
		taskJSON, _ := json.Marshal(task)
		var dialogue strings.Builder
		for _, message := range recent {
			fmt.Fprintf(&dialogue, "%s: %s\n", message.Role, message.Content)
		}
		temperature := .15
		response, generateErr := s.client.Generate(ctx, domain.ModelRequest{SystemPrompt: `Ты RAG-агент, который ведёт долгую задачу. Учитывай task memory, последние реплики и только подтверждённый контекст репозитория. Сохраняй принятые ограничения и определения. Ссылайся на контекст как [1], [2]. Если данных мало, скажи об этом.`, UserPrompt: "TASK MEMORY:\n" + string(taskJSON) + "\n\nRECENT DIALOGUE:\n" + dialogue.String() + "\nCURRENT USER MESSAGE:\n" + normalized + "\n\nRETRIEVED CONTEXT:\n" + formatRetrievedContext(retrieval.Improved), Temperature: &temperature, MaxTokens: 1000})
		if generateErr != nil {
			return domain.RAGChatExchange{}, generateErr
		}
		assistant.Content = strings.TrimSpace(response.Content)
		assistant.Usage = response.Usage
	}
	user, err = s.store.AppendRAGChatMessage(ctx, sessionID, user)
	if err != nil {
		return domain.RAGChatExchange{}, err
	}
	assistant, err = s.store.AppendRAGChatMessage(ctx, sessionID, assistant)
	if err != nil {
		return domain.RAGChatExchange{}, err
	}
	return domain.RAGChatExchange{User: user, Assistant: assistant, Task: task, Retrieved: retrieval.Improved}, nil
}

func (s *RAGService) ChatState(ctx context.Context, sessionID string) (domain.RAGChatState, error) {
	sessionID = strings.TrimSpace(sessionID)
	if !validRAGSession(sessionID) {
		return domain.RAGChatState{}, ErrInvalidRAGSession
	}
	messages, err := s.store.LoadRAGChat(ctx, sessionID)
	if err != nil {
		return domain.RAGChatState{}, err
	}
	task, _, err := s.store.LoadRAGTaskState(ctx, sessionID)
	if err != nil {
		return domain.RAGChatState{}, err
	}
	total := 0
	for _, message := range messages {
		total += message.Usage.TotalTokens
	}
	return domain.RAGChatState{SessionID: sessionID, Messages: messages, Task: task, TotalTokens: total}, nil
}
func (s *RAGService) ClearChat(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if !validRAGSession(sessionID) {
		return ErrInvalidRAGSession
	}
	return s.store.ClearRAGChat(ctx, sessionID)
}
func (s *RAGService) ChatScenarios() []domain.RAGChatScenario {
	return []domain.RAGChatScenario{
		{ID: "release", Name: "Релиз Knowledge Studio", Description: "12 реплик: цель, ограничения, определения и уточнения релиза.", Messages: []string{"Помоги подготовить релиз Knowledge Studio из этого репозитория.", "Релиз должен работать на текущем VPS и не раскрывать DeepSeek API key.", "Под MVP я понимаю страницы Дней 21–25 и рабочий health check.", "Сначала перечисли компоненты, которые уже есть в проекте.", "Бюджет — без новых платных сервисов.", "Какая роль у Caddy в этой схеме?", "Нужно сохранить старые лаборатории Дней 1–20.", "Составь порядок безопасной проверки перед деплоем.", "Добавь требование: при слабом retrieval агент не должен выдумывать ответ.", "Какие файлы подтверждают конфигурацию деплоя?", "Сведи ограничения и договорённости в короткий чек-лист.", "Подготовь итоговый план релиза с критериями готовности."}},
		{ID: "review", Name: "Техническое ревью RAG", Description: "11 реплик: исследование архитектуры и проверка принятых решений.", Messages: []string{"Проведи техническое ревью RAG-части проекта.", "Главная цель — проверяемые ответы с источниками.", "Термин evidence gate означает отказ от генерации ниже порога уверенности.", "Не предлагай внешнюю vector database.", "Сравни fixed и structural chunking по коду проекта.", "Какие метаданные сохраняются у каждого chunk?", "Проверь, как выполняется query rewrite.", "Нужно отдельно учитывать риск выдуманных цитат.", "Какая проверка гарантирует verbatim quote?", "Собери найденные риски, но не меняй исходное ограничение по базе данных.", "Дай итог ревью и три приоритетных улучшения."}},
	}
}

var ragSessionPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)

func validRAGSession(value string) bool { return ragSessionPattern.MatchString(value) }

var termPattern = regexp.MustCompile(`(?i)([\p{L}\d _-]{2,32})\s+(?:означает|это)\s+([^.!?\n]{2,100})`)

func updateRAGTaskState(state domain.RAGTaskState, sessionID, input string, now time.Time) domain.RAGTaskState {
	state.SessionID = sessionID
	if state.Constraints == nil {
		state.Constraints = []string{}
	}
	if state.Terms == nil {
		state.Terms = map[string]string{}
	}
	if strings.TrimSpace(state.Goal) == "" {
		state.Goal = input
	}
	lower := strings.ToLower(input)
	for _, marker := range []string{"долж", "нужно", "только", "не ", "огранич", "бюджет", "срок"} {
		if strings.Contains(lower, marker) {
			state.Constraints = appendUniqueBounded(state.Constraints, input, 8)
			break
		}
	}
	if match := termPattern.FindStringSubmatch(input); len(match) == 3 {
		state.Terms[strings.TrimSpace(match[1])] = strings.TrimSpace(match[2])
	}
	state.UpdatedAt = now
	return state
}
func appendUniqueBounded(values []string, value string, max int) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	values = append(values, value)
	if len(values) > max {
		values = values[len(values)-max:]
	}
	return values
}

func citationFrom(item domain.RetrievedChunk) domain.EvidenceCitation {
	quote := quoteExcerpt(item.Chunk.Content, 260)
	return domain.EvidenceCitation{Source: item.Chunk.Source, Section: item.Chunk.Section, ChunkID: item.Chunk.ID, Quote: quote, Score: maxFloat(item.RerankScore, item.SimilarityScore), QuoteValid: quote != "" && strings.Contains(item.Chunk.Content, quote)}
}
func quoteExcerpt(content string, limit int) string {
	trimmed := strings.TrimSpace(content)
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	cut := limit
	for cut > limit/2 && runes[cut] != '\n' && runes[cut] != '.' && runes[cut] != '!' && runes[cut] != '?' {
		cut--
	}
	if cut <= limit/2 {
		cut = limit
	}
	return strings.TrimSpace(string(runes[:cut]))
}
func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func (s *RAGService) loadChunks(ctx context.Context) ([]domain.KnowledgeChunk, error) {
	chunks, err := s.store.LoadKnowledgeChunks(ctx, domain.ChunkStrategyStructural)
	if err != nil {
		return nil, err
	}
	if len(chunks) > 0 {
		return chunks, nil
	}
	if _, err := s.indexer.Build(ctx); err != nil {
		return nil, fmt.Errorf("build missing knowledge index: %w", err)
	}
	return s.store.LoadKnowledgeChunks(ctx, domain.ChunkStrategyStructural)
}

func tokenSet(text string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, token := range tokenize(text) {
		result[token] = struct{}{}
	}
	return result
}
func lexicalOverlap(query, document map[string]struct{}) float64 {
	if len(query) == 0 {
		return 0
	}
	matches := 0
	for token := range query {
		if _, ok := document[token]; ok {
			matches++
		}
	}
	return float64(matches) / float64(len(query))
}

func (s *RAGService) validateQuestion(question string) (string, error) {
	normalized := strings.TrimSpace(question)
	if normalized == "" {
		return "", ErrEmptyRAGQuestion
	}
	if utf8.RuneCountInString(normalized) > s.maxRunes {
		return "", ErrRAGQuestionTooLong
	}
	return normalized, nil
}

func (s *RAGService) generate(ctx context.Context, system, user string) (domain.RAGAnswer, error) {
	started := time.Now()
	temperature := 0.1
	response, err := s.client.Generate(ctx, domain.ModelRequest{SystemPrompt: system, UserPrompt: user, Temperature: &temperature, MaxTokens: 900})
	if err != nil {
		return domain.RAGAnswer{}, err
	}
	return domain.RAGAnswer{Answer: strings.TrimSpace(response.Content), Model: response.Model, DurationMS: time.Since(started).Milliseconds(), Usage: response.Usage}, nil
}

func cosine(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	dot, leftNorm, rightNorm := 0.0, 0.0, 0.0
	for index := range left {
		dot += left[index] * right[index]
		leftNorm += left[index] * left[index]
		rightNorm += right[index] * right[index]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

func formatRetrievedContext(chunks []domain.RetrievedChunk) string {
	var builder strings.Builder
	for index, item := range chunks {
		fmt.Fprintf(&builder, "[%d] source=%s; section=%s; score=%.3f\n%s\n\n", index+1, item.Chunk.Source, item.Chunk.Section, item.SimilarityScore, item.Chunk.Content)
	}
	return strings.TrimSpace(builder.String())
}
