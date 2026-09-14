package follow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Store is the durable follows.json authority. Missing files start empty.
// Corrupt files are quarantined beside the path so they cannot take down Queue.
type Store struct {
	path string
	mu   sync.Mutex
	doc  Document
}

func PathForState(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "follows.json")
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("follow: empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	s := &Store{path: abs, doc: Document{Version: DocumentVersion, Follows: []Record{}}}
	data, err := os.ReadFile(abs)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("follow: read: %w", err)
	}
	doc, err := decodeDocument(data)
	if err != nil {
		bad := abs + ".bad"
		_ = os.Rename(abs, bad)
		return s, nil
	}
	s.doc = doc
	return s, nil
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) List() []Record {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, len(s.doc.Follows))
	for i := range s.doc.Follows {
		out[i] = CloneRecord(s.doc.Follows[i])
	}
	return out
}

func (s *Store) Get(id string) (Record, bool) {
	if s == nil {
		return Record{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.doc.Follows {
		if rec.ID == id {
			return CloneRecord(rec), true
		}
	}
	return Record{}, false
}

func (s *Store) ByPlaylist(playlistID string) (Record, bool) {
	if s == nil {
		return Record{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.doc.Follows {
		if rec.PlaylistID == playlistID {
			return CloneRecord(rec), true
		}
	}
	return Record{}, false
}

func (s *Store) Put(rec Record) error {
	if s == nil {
		return errors.New("follow: store is unavailable")
	}
	next := CloneRecord(rec)
	if next.UpdatedAt == "" {
		next.UpdatedAt = nowUTC()
	}
	if next.CreatedAt == "" {
		next.CreatedAt = next.UpdatedAt
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := CloneDocument(s.doc)
	replaced := false
	for i, existing := range doc.Follows {
		if existing.ID == next.ID {
			next.CreatedAt = existing.CreatedAt
			doc.Follows[i] = next
			replaced = true
			break
		}
		if existing.PlaylistID == next.PlaylistID && existing.ID != next.ID {
			return errors.New("this playlist is already followed")
		}
	}
	if !replaced {
		if len(doc.Follows) >= MaxFollows {
			return errors.New("too many followed playlists")
		}
		doc.Follows = append(doc.Follows, next)
	}
	return s.commitLocked(doc)
}

func (s *Store) Delete(id string) (Record, error) {
	if s == nil {
		return Record{}, errors.New("follow: store is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc := CloneDocument(s.doc)
	for i, rec := range doc.Follows {
		if rec.ID != id {
			continue
		}
		doc.Follows = append(doc.Follows[:i], doc.Follows[i+1:]...)
		if err := s.commitLocked(doc); err != nil {
			return Record{}, err
		}
		return CloneRecord(rec), nil
	}
	return Record{}, errors.New("that playlist is not followed")
}

func (s *Store) commitLocked(doc Document) error {
	if doc.Version == 0 {
		doc.Version = DocumentVersion
	}
	if err := ValidateDocument(doc); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("follow: marshal: %w", err)
	}
	if err := writeAtomic(s.path, data); err != nil {
		return err
	}
	s.doc = CloneDocument(doc)
	return nil
}

func decodeDocument(data []byte) (Document, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var doc Document
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Document{}, errors.New("follow: multiple JSON values")
		}
		return Document{}, err
	}
	if err := ValidateDocument(doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("follow: create directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".follows-")
	if err != nil {
		return fmt.Errorf("follow: create temp: %w", err)
	}
	name := tmp.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(name)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("follow: set temp permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("follow: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("follow: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("follow: close temp: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("follow: replace: %w", err)
	}
	keep = true
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("follow: set permissions: %w", err)
	}
	return nil
}
