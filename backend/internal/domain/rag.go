package domain

import "time"

type ChunkStrategy string

const (
	ChunkStrategyFixed      ChunkStrategy = "fixed"
	ChunkStrategyStructural ChunkStrategy = "structural"
)

type KnowledgeChunk struct {
	ID         string        `json:"chunkId"`
	Strategy   ChunkStrategy `json:"strategy"`
	Source     string        `json:"source"`
	Title      string        `json:"title"`
	Section    string        `json:"section"`
	Content    string        `json:"content"`
	Vector     []float64     `json:"-"`
	Dimensions int           `json:"dimensions"`
	CharCount  int           `json:"charCount"`
	CreatedAt  time.Time     `json:"createdAt"`
}

type IndexStrategyStats struct {
	Strategy       ChunkStrategy    `json:"strategy"`
	Documents      int              `json:"documents"`
	Chunks         int              `json:"chunks"`
	AverageChars   int              `json:"averageChars"`
	EmbeddingModel string           `json:"embeddingModel"`
	Dimensions     int              `json:"dimensions"`
	DurationMS     int64            `json:"durationMs"`
	Samples        []KnowledgeChunk `json:"samples"`
}

type KnowledgeIndexStatus struct {
	CorpusPath      string               `json:"corpusPath"`
	PagesEquivalent int                  `json:"pagesEquivalent"`
	BuiltAt         *time.Time           `json:"builtAt,omitempty"`
	Strategies      []IndexStrategyStats `json:"strategies"`
}

type RetrievedChunk struct {
	Chunk           KnowledgeChunk `json:"chunk"`
	SimilarityScore float64        `json:"similarityScore"`
	LexicalScore    float64        `json:"lexicalScore,omitempty"`
	RerankScore     float64        `json:"rerankScore,omitempty"`
}

type RAGAnswer struct {
	Answer     string `json:"answer"`
	Model      string `json:"model"`
	DurationMS int64  `json:"durationMs"`
	Usage      Usage  `json:"usage"`
}

type RAGComparison struct {
	Question   string           `json:"question"`
	WithoutRAG RAGAnswer        `json:"withoutRag"`
	WithRAG    RAGAnswer        `json:"withRag"`
	Retrieved  []RetrievedChunk `json:"retrieved"`
}

type ControlQuestion struct {
	ID              string   `json:"id"`
	Question        string   `json:"question"`
	ExpectedSources []string `json:"expectedSources"`
}

type RetrievalComparison struct {
	Question       string           `json:"question"`
	RewrittenQuery string           `json:"rewrittenQuery"`
	TopK           int              `json:"topK"`
	Threshold      float64          `json:"threshold"`
	Baseline       []RetrievedChunk `json:"baseline"`
	Candidates     []RetrievedChunk `json:"candidates"`
	Improved       []RetrievedChunk `json:"improved"`
	Dropped        int              `json:"dropped"`
	RewriteModel   string           `json:"rewriteModel"`
	RewriteUsage   Usage            `json:"rewriteUsage"`
	DurationMS     int64            `json:"durationMs"`
}

type EvidenceCitation struct {
	Source     string  `json:"source"`
	Section    string  `json:"section"`
	ChunkID    string  `json:"chunkId"`
	Quote      string  `json:"quote"`
	Score      float64 `json:"score"`
	QuoteValid bool    `json:"quoteValid"`
}

type GroundedAnswer struct {
	Status     string             `json:"status"`
	Question   string             `json:"question"`
	Answer     string             `json:"answer"`
	Confidence float64            `json:"confidence"`
	Threshold  float64            `json:"threshold"`
	Citations  []EvidenceCitation `json:"citations"`
	Model      string             `json:"model,omitempty"`
	Usage      Usage              `json:"usage"`
	DurationMS int64              `json:"durationMs"`
}

type EvidenceCheck struct {
	ID                    string            `json:"id"`
	Question              string            `json:"question"`
	ExpectedSources       []string          `json:"expectedSources"`
	TopSource             string            `json:"topSource,omitempty"`
	ExpectedSourceMatched bool              `json:"expectedSourceMatched"`
	AboveThreshold        bool              `json:"aboveThreshold"`
	QuoteValid            bool              `json:"quoteValid"`
	Score                 float64           `json:"score"`
	Citation              *EvidenceCitation `json:"citation,omitempty"`
}

type EvidenceReport struct {
	Threshold float64         `json:"threshold"`
	Checks    []EvidenceCheck `json:"checks"`
	Passed    int             `json:"passed"`
	Total     int             `json:"total"`
}

type RAGChatMessage struct {
	Sequence  int                `json:"sequence"`
	Role      string             `json:"role"`
	Content   string             `json:"content"`
	Citations []EvidenceCitation `json:"citations"`
	Usage     Usage              `json:"usage"`
	CreatedAt time.Time          `json:"createdAt"`
}

type RAGTaskState struct {
	SessionID   string            `json:"sessionId"`
	Goal        string            `json:"goal"`
	Constraints []string          `json:"constraints"`
	Terms       map[string]string `json:"terms"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type RAGChatState struct {
	SessionID   string           `json:"sessionId"`
	Messages    []RAGChatMessage `json:"messages"`
	Task        RAGTaskState     `json:"task"`
	TotalTokens int              `json:"totalTokens"`
}

type RAGChatExchange struct {
	User      RAGChatMessage   `json:"user"`
	Assistant RAGChatMessage   `json:"assistant"`
	Task      RAGTaskState     `json:"task"`
	Retrieved []RetrievedChunk `json:"retrieved"`
}

type RAGChatScenario struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Messages    []string `json:"messages"`
}
