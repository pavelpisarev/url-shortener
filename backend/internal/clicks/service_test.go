package clicks

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type addCall struct {
	code string
	day  time.Time
	n    int64
}

type fakeStore struct {
	addCalls []addCall
	addErr   func(code string) error
	stats    Stats
	statsErr error
}

func (f *fakeStore) AddClick(_ context.Context, code string, day time.Time, n int64) error {
	f.addCalls = append(f.addCalls, addCall{code: code, day: day, n: n})
	if f.addErr != nil {
		return f.addErr(code)
	}
	return nil
}

func (f *fakeStore) Stats(_ context.Context, code string) (Stats, error) {
	return f.stats, f.statsErr
}

type fakeQueue struct {
	calls []string
	err   error
}

func (f *fakeQueue) EnqueueClick(_ context.Context, code string) error {
	f.calls = append(f.calls, code)
	return f.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestServiceRecordUsesQueue(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	queue := &fakeQueue{}
	svc := NewService(store, queue)

	if err := svc.Record(context.Background(), "abc1234"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if len(queue.calls) != 1 || queue.calls[0] != "abc1234" {
		t.Fatalf("queue calls = %v, want [abc1234]", queue.calls)
	}
	if len(store.addCalls) != 0 {
		t.Fatalf("store calls = %d, want 0", len(store.addCalls))
	}
}

func TestServiceRecordFallsBackToStore(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	queue := &fakeQueue{err: errors.New("redis down")}
	svc := NewService(store, queue)

	before := time.Now().UTC()
	if err := svc.Record(context.Background(), "abc1234"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	after := time.Now().UTC()

	if len(store.addCalls) != 1 {
		t.Fatalf("store calls = %d, want 1", len(store.addCalls))
	}
	call := store.addCalls[0]
	if call.code != "abc1234" || call.n != 1 {
		t.Fatalf("store call = %+v", call)
	}
	if call.day.Location() != time.UTC {
		t.Fatalf("day location = %v, want UTC", call.day.Location())
	}
	if h, m, s := call.day.Clock(); h != 0 || m != 0 || s != 0 {
		t.Fatalf("day = %v, want midnight", call.day)
	}
	if call.day.Before(before.Truncate(24*time.Hour)) || call.day.After(after) {
		t.Fatalf("day = %v outside expected range", call.day)
	}
}

func TestServiceRecordWithoutQueue(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store, nil)

	if err := svc.Record(context.Background(), "abc1234"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if len(store.addCalls) != 1 || store.addCalls[0].code != "abc1234" {
		t.Fatalf("store calls = %+v", store.addCalls)
	}
}

func TestServiceRecordStoreError(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	store := &fakeStore{addErr: func(string) error { return boom }}
	svc := NewService(store, nil)

	if err := svc.Record(context.Background(), "abc1234"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestServiceStats(t *testing.T) {
	t.Parallel()

	want := Stats{Code: "abc1234", Total: 3, Days: []DayCount{{Day: "2026-01-02", Count: 3}}}
	store := &fakeStore{stats: want}
	svc := NewService(store, nil)

	got, err := svc.Stats(context.Background(), "abc1234")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if got.Code != want.Code || got.Total != want.Total || len(got.Days) != 1 || got.Days[0] != want.Days[0] {
		t.Fatalf("Stats = %+v, want %+v", got, want)
	}

	boom := errors.New("boom")
	store.statsErr = boom
	if _, err := svc.Stats(context.Background(), "abc1234"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}
