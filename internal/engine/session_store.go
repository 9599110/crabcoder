package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/crabcoder/crabcoder/pkg/model"
)

const (
	maxSessionFileSize = 256 * 1024 // 256KB
	maxRotatedFiles    = 3
	sessionFilePrefix  = "session-"
	sessionFileSuffix  = ".json"
)

// SessionRecord is the serializable form of a chat session.
type SessionRecord struct {
	ID        string          `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Messages  []model.Message `json:"messages"`
	Model     string          `json:"model"`
}

// SessionStore persists and loads chat sessions as JSON files.
type SessionStore struct {
	dataDir string
}

// NewSessionStore creates a session store rooted at dataDir.
func NewSessionStore(dataDir string) *SessionStore {
	return &SessionStore{dataDir: dataDir}
}

// Save writes a session record to disk with rotation.
func (s *SessionStore) Save(record *SessionRecord) error {
	if err := os.MkdirAll(s.dataDir, 0700); err != nil {
		return fmt.Errorf("session store: %w", err)
	}

	record.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("session marshal: %w", err)
	}

	if len(data) > maxSessionFileSize {
		// Truncate oldest messages to fit
		data = s.truncateRecord(record)
	}

	path := s.sessionPath(record.ID)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("session write: %w", err)
	}

	s.rotate(record.ID)
	return nil
}

// Load reads a session record from disk by ID.
func (s *SessionStore) Load(id string) (*SessionRecord, error) {
	path := s.sessionPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session %q not found", id)
		}
		return nil, fmt.Errorf("session read: %w", err)
	}

	var record SessionRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("session unmarshal: %w", err)
	}
	return &record, nil
}

// List returns all saved session IDs sorted by most recent first.
func (s *SessionStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []struct {
		id    string
		mtime time.Time
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, sessionFilePrefix) || !strings.HasSuffix(name, sessionFileSuffix) {
			continue
		}
		id := strings.TrimSuffix(strings.TrimPrefix(name, sessionFilePrefix), sessionFileSuffix)
		info, err := e.Info()
		if err != nil {
			continue
		}
		sessions = append(sessions, struct {
			id    string
			mtime time.Time
		}{id, info.ModTime()})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].mtime.After(sessions[j].mtime)
	})

	var ids []string
	for _, s := range sessions {
		ids = append(ids, s.id)
	}
	return ids, nil
}

// Delete removes a session file.
func (s *SessionStore) Delete(id string) error {
	path := s.sessionPath(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DataDir returns the store's data directory.
func (s *SessionStore) DataDir() string {
	return s.dataDir
}

func (s *SessionStore) sessionPath(id string) string {
	return filepath.Join(s.dataDir, sessionFilePrefix+id+sessionFileSuffix)
}

// rotate keeps at most maxRotatedFiles versions of a session.
func (s *SessionStore) rotate(id string) {
	for i := maxRotatedFiles; i >= 1; i-- {
		oldPath := s.rotatedPath(id, i)
		if i == maxRotatedFiles {
			os.Remove(oldPath)
			continue
		}
		newPath := s.rotatedPath(id, i+1)
		os.Rename(oldPath, newPath)
	}
	// Rotate current to .1
	os.Rename(s.sessionPath(id), s.rotatedPath(id, 1))
}

func (s *SessionStore) rotatedPath(id string, n int) string {
	return filepath.Join(s.dataDir, fmt.Sprintf("%s%s.%d%s", sessionFilePrefix, id, n, sessionFileSuffix))
}

// truncateRecord removes oldest non-system messages until the serialized size fits.
func (s *SessionStore) truncateRecord(record *SessionRecord) []byte {
	for len(record.Messages) > 2 {
		data, _ := json.MarshalIndent(record, "", "  ")
		if len(data) <= maxSessionFileSize {
			return data
		}
		// Remove the oldest non-system message
		for i, m := range record.Messages {
			if m.Role != model.RoleSystem {
				record.Messages = append(record.Messages[:i], record.Messages[i+1:]...)
				break
			}
		}
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	return data
}
