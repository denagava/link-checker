package main

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	StatusAvailable    = "available"
	StatusNotAvailable = "not available"

	// DefaultConcurrency limits simultaneous outgoing requests so a single
	// large batch cannot exhaust sockets or overload remote hosts.
	DefaultConcurrency = 20
)

type LinkChecker struct {
	client      *http.Client
	concurrency int
}

func NewLinkChecker(concurrency int) *LinkChecker {
	if concurrency < 1 {
		concurrency = 1
	}
	return &LinkChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
			},
		},
		concurrency: concurrency,
	}
}

// normalizeURL adds an http:// scheme to bare domains like "google.com".
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return "http://" + raw
}

// CheckLink reports whether url answers with a non-error HTTP status.
func (lc *LinkChecker) CheckLink(ctx context.Context, url string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalizeURL(url), nil)
	if err != nil {
		return StatusNotAvailable
	}
	resp, err := lc.client.Do(req)
	if err != nil {
		return StatusNotAvailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusBadRequest {
		return StatusAvailable
	}
	return StatusNotAvailable
}

// CheckLinks checks all urls concurrently, at most lc.concurrency at a time.
func (lc *LinkChecker) CheckLinks(ctx context.Context, urls []string) map[string]string {
	results := make(map[string]string, len(urls))
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, lc.concurrency)
	)

	for _, u := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			status := StatusNotAvailable
			select {
			case sem <- struct{}{}:
				status = lc.CheckLink(ctx, u)
				<-sem
			case <-ctx.Done():
			}

			mu.Lock()
			results[u] = status
			mu.Unlock()
		}(u)
	}
	wg.Wait()
	return results
}
