package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type LinkSet struct {
	Links     map[string]string `json:"links"`
	Timestamp time.Time         `json:"timestamp"`
	LinksNum  int               `json:"links_num"`
}

type storageFile struct {
	LinksSets map[int]*LinkSet `json:"links_sets"`
	NextID    int              `json:"next_id"`
}

// Storage keeps checked link sets in memory and persists them to a JSON file.
type Storage struct {
	mu     sync.RWMutex
	sets   map[int]*LinkSet
	nextID int
	path   string
}

func NewStorage(path string) (*Storage, error) {
	s := &Storage{sets: make(map[int]*LinkSet), nextID: 1, path: path}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var f storageFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.LinksSets != nil {
		s.sets = f.LinksSets
	}
	if f.NextID > 0 {
		s.nextID = f.NextID
	}
	return s, nil
}

// SaveLinkSet stores links under a new ID and persists the whole state.
func (s *Storage) SaveLinkSet(links map[string]string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	s.sets[id] = &LinkSet{Links: links, Timestamp: time.Now(), LinksNum: id}
	s.nextID++

	if err := s.persistLocked(); err != nil {
		return id, err
	}
	return id, nil
}

// GetLinkSets returns the sets that exist among ids.
func (s *Storage) GetLinkSets(ids []int) map[int]*LinkSet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[int]*LinkSet, len(ids))
	for _, id := range ids {
		if set, ok := s.sets[id]; ok {
			result[id] = set
		}
	}
	return result
}

// persistLocked writes state atomically (temp file + rename). Caller holds mu.
func (s *Storage) persistLocked() error {
	data, err := json.MarshalIndent(storageFile{LinksSets: s.sets, NextID: s.nextID}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal storage: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}
