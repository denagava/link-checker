package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNormalizeURL(t *testing.T) {
	tests := map[string]string{
		"google.com":          "http://google.com",
		"  yandex.ru ":        "http://yandex.ru",
		"http://example.com":  "http://example.com",
		"https://example.com": "https://example.com",
	}
	for in, want := range tests {
		if got := normalizeURL(in); got != want {
			t.Errorf("normalizeURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckLink(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer ok.Close()
	notFound := httptest.NewServer(http.NotFoundHandler())
	defer notFound.Close()
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer broken.Close()

	lc := NewLinkChecker(DefaultConcurrency)
	ctx := context.Background()

	cases := map[string]string{
		ok.URL:                StatusAvailable,
		notFound.URL:          StatusNotAvailable,
		broken.URL:            StatusNotAvailable,
		"http://127.0.0.1:1":  StatusNotAvailable, // nothing listens there
		"http://bad host.com": StatusNotAvailable, // invalid URL
	}
	for url, want := range cases {
		if got := lc.CheckLink(ctx, url); got != want {
			t.Errorf("CheckLink(%s) = %q, want %q", url, got, want)
		}
	}
}

func TestCheckLinks_RespectsConcurrencyLimit(t *testing.T) {
	const limit = 3
	var current, peak atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := current.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		current.Add(-1)
	}))
	defer srv.Close()

	urls := make([]string, 15)
	for i := range urls {
		urls[i] = srv.URL + "/" + string(rune('a'+i))
	}

	results := NewLinkChecker(limit).CheckLinks(context.Background(), urls)

	if len(results) != len(urls) {
		t.Fatalf("got %d results, want %d", len(results), len(urls))
	}
	for u, s := range results {
		if s != StatusAvailable {
			t.Errorf("%s: %s", u, s)
		}
	}
	if p := peak.Load(); p > limit {
		t.Fatalf("peak concurrency %d exceeds limit %d", p, limit)
	}
}

func TestCheckLinks_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := NewLinkChecker(2).CheckLinks(ctx, []string{"a.example", "b.example"})
	for u, s := range results {
		if s != StatusNotAvailable {
			t.Errorf("%s: got %q after cancel", u, s)
		}
	}
}
