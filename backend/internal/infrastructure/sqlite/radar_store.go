package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func (s *ConversationStore) SaveDigest(ctx context.Context, digest domain.ScheduledDigest) error {
	raw, err := json.Marshal(digest.Result)
	if err != nil {
		return fmt.Errorf("encode radar digest: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO radar_digests
		(digest_id, trigger_type, query_text, result_json, created_at) VALUES (?, ?, ?, ?, ?)`,
		digest.ID, digest.Trigger, digest.Query, string(raw), digest.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save radar digest: %w", err)
	}
	return nil
}

func (s *ConversationStore) ListDigests(ctx context.Context, limit int) ([]domain.ScheduledDigest, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx, `SELECT digest_id, trigger_type, query_text, result_json, created_at
		FROM radar_digests ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list radar digests: %w", err)
	}
	defer rows.Close()
	digests := []domain.ScheduledDigest{}
	for rows.Next() {
		var digest domain.ScheduledDigest
		var raw, createdAt string
		if err := rows.Scan(&digest.ID, &digest.Trigger, &digest.Query, &raw, &createdAt); err != nil {
			return nil, fmt.Errorf("scan radar digest: %w", err)
		}
		if err := json.Unmarshal([]byte(raw), &digest.Result); err != nil {
			return nil, fmt.Errorf("decode radar digest: %w", err)
		}
		digest.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse radar digest time: %w", err)
		}
		digests = append(digests, digest)
	}
	return digests, rows.Err()
}

func (s *ConversationStore) SaveRadarReport(ctx context.Context, report domain.RadarReport) error {
	brief, err := json.Marshal(report.Brief)
	if err != nil {
		return fmt.Errorf("encode radar report: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO radar_reports(report_id, query_text, brief_json, source_count, created_at) VALUES(?, ?, ?, ?, ?)`,
		report.ID, report.Query, string(brief), report.SourceCount, report.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save radar report: %w", err)
	}
	return nil
}
