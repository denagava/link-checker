package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorage_SaveAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")

	s, err := NewStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	id1, err := s.SaveLinkSet(map[string]string{"a.com": StatusAvailable})
	if err != nil {
		t.Fatal(err)
	}
	id2, _ := s.SaveLinkSet(map[string]string{"b.com": StatusNotAvailable})
	if id1 != 1 || id2 != 2 {
		t.Fatalf("ids = %d, %d; want 1, 2", id1, id2)
	}

	// A new instance must see persisted data and continue numbering.
	s2, err := NewStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	got := s2.GetLinkSets([]int{1, 2, 99})
	if len(got) != 2 {
		t.Fatalf("got %d sets, want 2", len(got))
	}
	if got[2].Links["b.com"] != StatusNotAvailable {
		t.Errorf("set 2 = %+v", got[2])
	}
	if id3, _ := s2.SaveLinkSet(map[string]string{}); id3 != 3 {
		t.Errorf("next id after reload = %d, want 3", id3)
	}
}

func TestStorage_CorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStorage(path); err == nil {
		t.Fatal("expected error for corrupt file")
	}
}
