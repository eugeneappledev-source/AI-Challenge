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
