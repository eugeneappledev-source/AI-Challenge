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
