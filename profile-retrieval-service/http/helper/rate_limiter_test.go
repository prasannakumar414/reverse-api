package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterRejectsRepeatedClientRequests(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(RateLimitConfig{
		Requests: 1,
		Window:   time.Minute,
		Now: func() time.Time {
			return now
		},
	})
	handler := limiter.Wrap(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	first := httptest.NewRequest(http.MethodGet, "/profiles/retrieve", nil)
	first.RemoteAddr = "203.0.113.10:12345"
	firstResponse := httptest.NewRecorder()
	handler(firstResponse, first)

	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first response code = %d", firstResponse.Code)
	}

	second := httptest.NewRequest(http.MethodGet, "/profiles/retrieve", nil)
	second.RemoteAddr = "203.0.113.10:54321"
	secondResponse := httptest.NewRecorder()
	handler(secondResponse, second)

	if secondResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("second response code = %d", secondResponse.Code)
	}
	if retryAfter := secondResponse.Header().Get("Retry-After"); retryAfter != "60" {
		t.Fatalf("Retry-After = %q", retryAfter)
	}
}

func TestRateLimiterAllowsRequestsAfterWindowResets(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(RateLimitConfig{
		Requests: 1,
		Window:   time.Minute,
		Now: func() time.Time {
			return now
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/profiles/retrieve", nil)
	request.RemoteAddr = "203.0.113.10:12345"

	if _, ok := limiter.Allow(request); !ok {
		t.Fatal("first request was rejected")
	}
	if _, ok := limiter.Allow(request); ok {
		t.Fatal("second request was allowed before reset")
	}

	now = now.Add(time.Minute)
	if _, ok := limiter.Allow(request); !ok {
		t.Fatal("request was rejected after reset")
	}
}
