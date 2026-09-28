package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type fakeResearcher struct{}

func (fakeResearcher) Research(_ context.Context, request string) (domain.BusinessResearchResult, error) {
	return domain.BusinessResearchResult{Request: request, Advice: domain.BusinessAdvice{Summary: "Daily summary"}}, nil
}

type memoryDigestStore struct{ digests []domain.ScheduledDigest }

func (s *memoryDigestStore) SaveDigest(_ context.Context, digest domain.ScheduledDigest) error {
	s.digests = append([]domain.ScheduledDigest{digest}, s.digests...)
	return nil
}

func (s *memoryDigestStore) ListDigests(_ context.Context, _ int) ([]domain.ScheduledDigest, error) {
	return s.digests, nil
}

func TestScheduledDigestPersistsCronResult(t *testing.T) {
	store := &memoryDigestStore{}
	service := application.NewScheduledDigestService(fakeResearcher{}, store, "0 8 * * *", "UTC")
	digest, err := service.Run(context.Background(), "cron")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if digest.Trigger != "cron" || len(store.digests) != 1 {
		t.Fatalf("digest was not persisted: %+v", digest)
	}
	dashboard, err := service.Dashboard(context.Background())
	if err != nil || dashboard.Schedule.LastStatus != "completed" || len(dashboard.Digests) != 1 {
		t.Fatalf("unexpected dashboard: %+v, error: %v", dashboard, err)
	}
	if dashboard.Schedule.NextRunAt.Before(time.Now().UTC()) {
		t.Fatalf("next run must be in the future: %s", dashboard.Schedule.NextRunAt)
	}
}
