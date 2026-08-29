package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRequestPacerSpacesRequests(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	var waits []time.Duration
	pacer := NewRequestPacer(2 * time.Second)
	pacer.now = func() time.Time {
		return now
	}
	pacer.after = func(wait time.Duration) <-chan time.Time {
		waits = append(waits, wait)
		ch := make(chan time.Time, 1)
		ch <- now.Add(wait)
		return ch
	}

	if err := pacer.Wait(context.Background()); err != nil {
		t.Fatalf("first wait returned error: %v", err)
	}
	if err := pacer.Wait(context.Background()); err != nil {
		t.Fatalf("second wait returned error: %v", err)
	}
	if err := pacer.Wait(context.Background()); err != nil {
		t.Fatalf("third wait returned error: %v", err)
	}

	expected := []time.Duration{2 * time.Second, 4 * time.Second}
	if !reflect.DeepEqual(waits, expected) {
		t.Fatalf("waits = %#v", waits)
	}
}

func TestRequestPacerHonorsContextCancellation(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	pacer := NewRequestPacer(time.Minute)
	pacer.now = func() time.Time {
		return now
	}
	pacer.after = func(time.Duration) <-chan time.Time {
		return make(chan time.Time)
	}

	if err := pacer.Wait(context.Background()); err != nil {
		t.Fatalf("first wait returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := pacer.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("second wait error = %v", err)
	}
}
