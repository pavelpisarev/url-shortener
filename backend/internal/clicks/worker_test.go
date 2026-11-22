package clicks

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakePopper struct {
	calls []int
	popFn func(ctx context.Context, max int) ([]string, error)
}

func (f *fakePopper) PopClicks(ctx context.Context, max int) ([]string, error) {
	f.calls = append(f.calls, max)
	return f.popFn(ctx, max)
}

func TestWorkerFlushAggregatesCodes(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	worker := NewWorker(nil, store, testLogger())

	worker.flush(context.Background(), []string{"a", "b", "a", "a", "c", "b"})

	if len(store.addCalls) != 3 {
		t.Fatalf("store calls = %d, want 3", len(store.addCalls))
	}
	want := map[string]int64{"a": 3, "b": 2, "c": 1}
	for _, call := range store.addCalls {
		if want[call.code] != call.n {
			t.Fatalf("call %+v, want count %d", call, want[call.code])
		}
		if call.day.Location() != time.UTC {
			t.Fatalf("day location = %v, want UTC", call.day.Location())
		}
	}
}

func TestWorkerFlushContinuesAfterStoreError(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		addErr: func(code string) error {
			if code == "b" {
				return errors.New("boom")
			}
			return nil
		},
	}
	worker := NewWorker(nil, store, testLogger())

	worker.flush(context.Background(), []string{"a", "b", "c"})

	if len(store.addCalls) != 3 {
		t.Fatalf("store calls = %d, want 3 attempts", len(store.addCalls))
	}
}

func TestWorkerFlushEmpty(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	worker := NewWorker(nil, store, testLogger())

	worker.flush(context.Background(), nil)

	if len(store.addCalls) != 0 {
		t.Fatalf("store calls = %d, want 0", len(store.addCalls))
	}
}

func TestWorkerRunProcessesBatchAndStopsOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &fakeStore{}
	popper := &fakePopper{}
	popper.popFn = func(ctx context.Context, max int) ([]string, error) {
		if len(popper.calls) == 1 {
			return []string{"a", "a", "b"}, nil
		}
		cancel()
		return nil, context.Canceled
	}

	NewWorker(popper, store, testLogger()).Run(ctx)

	if len(popper.calls) != 2 {
		t.Fatalf("pop calls = %d, want 2", len(popper.calls))
	}
	if popper.calls[0] != 500 {
		t.Fatalf("batch size = %d, want 500", popper.calls[0])
	}
	if len(store.addCalls) != 2 {
		t.Fatalf("store calls = %d, want 2", len(store.addCalls))
	}
	total := int64(0)
	for _, call := range store.addCalls {
		total += call.n
	}
	if total != 3 {
		t.Fatalf("total clicks = %d, want 3", total)
	}
}

func TestWorkerRunStopsWhenContextAlreadyCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	popper := &fakePopper{
		popFn: func(context.Context, int) ([]string, error) {
			t.Fatal("PopClicks called with canceled context")
			return nil, nil
		},
	}
	NewWorker(popper, &fakeStore{}, testLogger()).Run(ctx)

	if len(popper.calls) != 0 {
		t.Fatalf("pop calls = %d, want 0", len(popper.calls))
	}
}
