package engine

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/crabcoder/crabcoder/pkg/model"
)

const (
	maxSessionFileSize = 256 * 1024 // 256KB, rotate above this
	maxRotatedFiles    = 3
	sessionFileSuffix  = ".json"
	tmpFilePrefix      = ".tmp-"
	rotFilePrefix      = ".rot-"
)

var sessionIDCounter atomic.Int64

// GenerateSessionID creates a unique session ID (matches crab-code format).
func GenerateSessionID() string {
	millis := time.Now().UnixMilli()
	counter := sessionIDCounter.Add(1)
	return fmt.Sprintf("session-%d-%d", millis, counter)
}

// SessionRecord is the serializable form of a chat session.
type SessionRecord struct {
	ID            string          `json:"id"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Messages      []model.Message `json:"messages"`
	Model         string          `json:"model"`
	WorkspaceRoot string          `json:"workspace_root,omitempty"`
}

// SessionStore persists and loads chat sessions as JSON files,
// organized per-workspace under <dataDir>/<workspace_hash>/.
type SessionStore struct {
	dataDir       string
	workspaceRoot string
}

// NewSessionStore creates a session store rooted at dataDir for the given workspace.
func NewSessionStore(dataDir string) *SessionStore {
	cwd, _ := os.Getwd()
	return &SessionStore{
		dataDir:       dataDir,
		workspaceRoot: cwd,
	}
}

// sessionDir returns the per-workspace session directory.
func (s *SessionStore) sessionDir() string {
	return filepath.Join(s.dataDir, workspaceFingerprint(s.workspaceRoot))
}

// Save writes a session record to disk with atomic write and rotation.
func (s *SessionStore) Save(record *SessionRecord) error {
	dir := s.sessionDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("session store: %w", err)
	}

	record.UpdatedAt = time.Now()
	if record.WorkspaceRoot == "" {
		record.WorkspaceRoot = s.workspaceRoot
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("session marshal: %w", err)
	}

	if len(data) > maxSessionFileSize {
		data = s.truncateRecord(record)
	}

	targetPath := s.sessionPath(record.ID)

	// Rotate existing before writing new
	if _, err := os.Stat(targetPath); err == nil {
		s.rotate(record.ID)
	}

	// Atomic write: tmp file then rename
	tmpPath := filepath.Join(dir, fmt.Sprintf("%s%d-%d%s", tmpFilePrefix, time.Now().UnixMilli(), sessionIDCounter.Add(1), sessionFileSuffix))
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("session write: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("session atomic rename: %w", err)
	}

	return nil
}

// Load reads a session record by ID or alias ("latest", "last").
func (s *SessionStore) Load(ref string) (*SessionRecord, error) {
	if ref == "latest" || ref == "last" || ref == "recent" {
		latest, err := s.Latest()
		if err != nil {
			return nil, err
		}
		if latest == nil {
			return nil, fmt.Errorf("no saved sessions found")
		}
		return latest, nil
	}

	// Try exact ID match
	path := s.sessionPath(ref)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try prefix match
			ids, listErr := s.List()
			if listErr != nil {
				return nil, fmt.Errorf("session %q not found", ref)
			}
			for _, id := range ids {
				if strings.HasPrefix(id, ref) {
					return s.Load(id)
				}
			}
			return nil, fmt.Errorf("session %q not found", ref)
		}
		return nil, fmt.Errorf("session read: %w", err)
	}

	var record SessionRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("session unmarshal: %w", err)
	}
	return &record, nil
}

// Latest returns the most recently modified session, or nil if none exist.
func (s *SessionStore) Latest() (*SessionRecord, error) {
	ids, err := s.List()
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	return s.Load(ids[0])
}

// List returns all saved session IDs sorted by most recent first.
func (s *SessionStore) List() ([]string, error) {
	dir := s.sessionDir()
	entries, err := os.ReadDir(dir)
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
		// Skip tmp and rotated files
		if strings.HasPrefix(name, tmpFilePrefix) || strings.HasPrefix(name, rotFilePrefix) {
			continue
		}
		if !strings.HasSuffix(name, sessionFileSuffix) {
			continue
		}
		id := strings.TrimSuffix(name, sessionFileSuffix)
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
	for _, se := range sessions {
		ids = append(ids, se.id)
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
	return filepath.Join(s.sessionDir(), id+sessionFileSuffix)
}

// rotate keeps at most maxRotatedFiles rotated versions.
func (s *SessionStore) rotate(id string) {
	targetPath := s.sessionPath(id)
	ts := time.Now().UnixMilli()
	for i := maxRotatedFiles; i >= 1; i-- {
		oldPath := s.rotatedPath(id, i)
		if i == maxRotatedFiles {
			os.Remove(oldPath)
			continue
		}
		newPath := s.rotatedPath(id, i+1)
		os.Rename(oldPath, newPath)
	}
	// Rotate current to .rot.<ts>.1
	os.Rename(targetPath, s.rotatedPath(id, 1))
	// Touch the latest rotated file to record rotation time
	os.Chtimes(s.rotatedPath(id, 1), time.Now(), time.Now())
	_ = ts
}

func (s *SessionStore) rotatedPath(id string, n int) string {
	return filepath.Join(s.sessionDir(), fmt.Sprintf("%s%d.%d%s", rotFilePrefix, time.Now().UnixMilli(), n, sessionFileSuffix))
}

// truncateRecord removes oldest non-system messages until the serialized size fits.
func (s *SessionStore) truncateRecord(record *SessionRecord) []byte {
	for len(record.Messages) > 2 {
		data, _ := json.MarshalIndent(record, "", "  ")
		if len(data) <= maxSessionFileSize {
			return data
		}
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

// workspaceFingerprint returns a 16-char hex string of the FNV-1a 64-bit hash
// of the canonical workspace path, matching crab-code's approach.
func workspaceFingerprint(workspaceRoot string) string {
	h := fnv.New64a()
	h.Write([]byte(workspaceRoot))
	return fmt.Sprintf("%016x", h.Sum64())
}
