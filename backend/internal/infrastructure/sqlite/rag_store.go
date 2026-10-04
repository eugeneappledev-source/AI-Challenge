package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func (s *ConversationStore) ReplaceKnowledgeChunks(ctx context.Context, strategy domain.ChunkStrategy, chunks []domain.KnowledgeChunk) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace knowledge chunks: %w", err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM rag_chunks WHERE strategy = ?`, strategy); err != nil {
		return fmt.Errorf("clear knowledge chunks: %w", err)
	}
	statement, err := tx.PrepareContext(ctx, `INSERT INTO rag_chunks(chunk_id,strategy,source,title,section_name,content,vector_json,dimensions,char_count,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, chunk := range chunks {
		vector, marshalErr := json.Marshal(chunk.Vector)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = statement.ExecContext(ctx, chunk.ID, chunk.Strategy, chunk.Source, chunk.Title, chunk.Section, chunk.Content, string(vector), chunk.Dimensions, chunk.CharCount, chunk.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("insert knowledge chunk: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit knowledge chunks: %w", err)
	}
	return nil
}

func (s *ConversationStore) KnowledgeIndexStatus(ctx context.Context, corpusPath string) (domain.KnowledgeIndexStatus, error) {
	status := domain.KnowledgeIndexStatus{CorpusPath: corpusPath, Strategies: []domain.IndexStrategyStats{}}
	for _, strategy := range []domain.ChunkStrategy{domain.ChunkStrategyFixed, domain.ChunkStrategyStructural} {
		var stats domain.IndexStrategyStats
		stats.Strategy = strategy
		var builtAt string
		err := s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT source),COUNT(*),CAST(COALESCE(AVG(char_count),0) AS INTEGER),COALESCE(MAX(created_at),'') FROM rag_chunks WHERE strategy=?`, strategy).Scan(&stats.Documents, &stats.Chunks, &stats.AverageChars, &builtAt)
		if err != nil {
			return status, fmt.Errorf("load index stats: %w", err)
		}
		stats.EmbeddingModel = "local-feature-hash-v1"
		stats.Dimensions = 256
		rows, err := s.db.QueryContext(ctx, `SELECT chunk_id,source,title,section_name,content,dimensions,char_count,created_at FROM rag_chunks WHERE strategy=? ORDER BY source,chunk_id LIMIT 3`, strategy)
		if err != nil {
			return status, err
		}
		for rows.Next() {
			var chunk domain.KnowledgeChunk
			var created string
			chunk.Strategy = strategy
			if err := rows.Scan(&chunk.ID, &chunk.Source, &chunk.Title, &chunk.Section, &chunk.Content, &chunk.Dimensions, &chunk.CharCount, &created); err != nil {
				rows.Close()
				return status, err
			}
			chunk.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
			runes := []rune(chunk.Content)
			if len(runes) > 360 {
				chunk.Content = string(runes[:360]) + "…"
			}
			stats.Samples = append(stats.Samples, chunk)
		}
		rows.Close()
		status.Strategies = append(status.Strategies, stats)
		if builtAt != "" {
			parsed, _ := time.Parse(time.RFC3339Nano, builtAt)
			if status.BuiltAt == nil || parsed.After(*status.BuiltAt) {
				status.BuiltAt = &parsed
			}
		}
	}
	var totalChars int
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(char_count),0) FROM rag_chunks WHERE strategy=?`, domain.ChunkStrategyStructural).Scan(&totalChars)
	status.PagesEquivalent = totalChars / 1800
	return status, nil
}
