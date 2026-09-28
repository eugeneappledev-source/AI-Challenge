package application

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

const scheduledDigestQuery = "Собери главные новости о стартапах, AI-продуктах и инструментах для бизнеса. Выдели проверяемые возможности для небольшого цифрового продукта."

type BusinessResearcher interface {
	Research(ctx context.Context, request string) (domain.BusinessResearchResult, error)
}

type DigestStore interface {
	SaveDigest(ctx context.Context, digest domain.ScheduledDigest) error
	ListDigests(ctx context.Context, limit int) ([]domain.ScheduledDigest, error)
}

type ScheduledDigestService struct {
	researcher BusinessResearcher
	store      DigestStore
	cron       string
	timezone   string
	now        func() time.Time
	counter    atomic.Uint64
}

func NewScheduledDigestService(researcher BusinessResearcher, store DigestStore, cron, timezone string) *ScheduledDigestService {
	return &ScheduledDigestService{researcher: researcher, store: store, cron: cron, timezone: timezone, now: time.Now}
}

func (s *ScheduledDigestService) Run(ctx context.Context, trigger string) (domain.ScheduledDigest, error) {
	trigger = strings.TrimSpace(trigger)
	if trigger != "cron" {
		trigger = "manual"
	}
	result, err := s.researcher.Research(ctx, scheduledDigestQuery)
	if err != nil {
		return domain.ScheduledDigest{}, fmt.Errorf("run scheduled research: %w", err)
	}
	now := s.now().UTC()
	digest := domain.ScheduledDigest{
		ID: fmt.Sprintf("digest-%d-%d", now.Unix(), s.counter.Add(1)), Trigger: trigger,
		Query: scheduledDigestQuery, Result: result, CreatedAt: now,
	}
	if err := s.store.SaveDigest(ctx, digest); err != nil {
		return domain.ScheduledDigest{}, err
	}
	return digest, nil
}

func (s *ScheduledDigestService) Dashboard(ctx context.Context) (domain.DigestDashboard, error) {
	digests, err := s.store.ListDigests(ctx, 10)
	if err != nil {
		return domain.DigestDashboard{}, err
	}
	now := s.now().UTC()
	nextRun := nextDailyRun(now, 8)
	schedule := domain.RadarSchedule{Cron: s.cron, Timezone: s.timezone, Enabled: true, NextRunAt: nextRun, LastStatus: "waiting"}
	if len(digests) > 0 {
		schedule.LastRunAt = &digests[0].CreatedAt
		schedule.LastStatus = "completed"
	}
	return domain.DigestDashboard{Schedule: schedule, Digests: digests}, nil
}

func nextDailyRun(now time.Time, hour int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
