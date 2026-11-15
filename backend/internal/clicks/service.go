package clicks

import (
	"context"
	"time"

	"urlshortener/internal/platform/obs"
)

type Store interface {
	AddClick(ctx context.Context, code string, day time.Time, n int64) error
	Stats(ctx context.Context, code string) (Stats, error)
}

type Queue interface {
	EnqueueClick(ctx context.Context, code string) error
}

type Service struct {
	store Store
	queue Queue
}

func NewService(store Store, queue Queue) *Service {
	return &Service{store: store, queue: queue}
}

func (s *Service) Record(ctx context.Context, code string) error {
	if s.queue != nil {
		if err := s.queue.EnqueueClick(ctx, code); err == nil {
			obs.IncClickEnqueued()
			return nil
		}
	}
	return s.store.AddClick(ctx, code, today(), 1)
}

func (s *Service) Stats(ctx context.Context, code string) (Stats, error) {
	return s.store.Stats(ctx, code)
}

func today() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
