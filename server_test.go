package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) http.Handler {
	t.Helper()
	st, err := NewStorage(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(st, NewLinkChecker(DefaultConcurrency), NewPDFGenerator()).Routes()
}

func post(h http.Handler, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
	return rec
}

func TestCheckThenReport(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer target.Close()
	app := newTestApp(t)

	body, _ := json.Marshal(map[string]any{"links": []string{target.URL, "http://127.0.0.1:1"}})
	rec := post(app, "/check", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("/check status = %d, body = %s", rec.Code, rec.Body)
	}
	var resp checkResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.LinksNum != 1 {
		t.Errorf("links_num = %d, want 1", resp.LinksNum)
	}
	if resp.Links[target.URL] != StatusAvailable || resp.Links["http://127.0.0.1:1"] != StatusNotAvailable {
		t.Errorf("links = %v", resp.Links)
	}

	rec = post(app, "/report", `{"links_list":[1]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("/report status = %d, body = %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q", ct)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
		t.Error("response is not a PDF")
	}
}

func TestValidation(t *testing.T) {
	app := newTestApp(t)
	many, _ := json.Marshal(map[string]any{"links": make([]string, maxLinksPerRequest+1)})

	cases := []struct {
		name, path, body string
		want             int
	}{
		{"check/invalid_json", "/check", "{", http.StatusBadRequest},
		{"check/empty", "/check", `{"links":[]}`, http.StatusBadRequest},
		{"check/too_many", "/check", string(many), http.StatusBadRequest},
		{"report/invalid_json", "/report", "{", http.StatusBadRequest},
		{"report/empty", "/report", `{"links_list":[]}`, http.StatusBadRequest},
		{"report/not_found", "/report", `{"links_list":[42]}`, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := post(app, tc.path, tc.body); rec.Code != tc.want {
				t.Errorf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	app := newTestApp(t)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/check", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
