package clicks

import (
	"context"
	"log/slog"
	"time"
)

type Popper interface {
	PopClicks(ctx context.Context, max int) ([]string, error)
}

type Worker struct {
	queue Popper
	store Store
	log   *slog.Logger
	batch int
}

func NewWorker(queue Popper, store Store, log *slog.Logger) *Worker {
	return &Worker{queue: queue, store: store, log: log, batch: 500}
}

func (w *Worker) Run(ctx context.Context) {
	w.log.Info("click worker started")
	defer w.log.Info("click worker stopped")

	for {
		if ctx.Err() != nil {
			return
		}

		codes, err := w.queue.PopClicks(ctx, w.batch)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			w.log.Error("pop clicks", "err", err)
			time.Sleep(time.Second)
			continue
		}
		if len(codes) == 0 {
			continue
		}
		w.flush(ctx, codes)
	}
}

func (w *Worker) flush(ctx context.Context, codes []string) {
	counts := make(map[string]int64, len(codes))
	for _, code := range codes {
		counts[code]++
	}

	day := today()
	for code, n := range counts {
		if err := w.store.AddClick(ctx, code, day, n); err != nil {
			w.log.Error("aggregate clicks", "code", code, "err", err)
		}
	}
}
