package security

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemIsolationMode defines the sandbox filesystem policy.
type FilesystemIsolationMode string

const (
	// FsOff — no filesystem restrictions.
	FsOff FilesystemIsolationMode = "off"
	// FsWorkspaceOnly — restrict file access to the workspace root.
	FsWorkspaceOnly FilesystemIsolationMode = "workspace"
	// FsAllowList — only paths in the allow list are accessible.
	FsAllowList FilesystemIsolationMode = "allowlist"
)

// Sandbox controls execution isolation for tool invocations.
type Sandbox struct {
	Network    bool
	Filesystem FilesystemIsolationMode
	WorkDir    string
	AllowPaths []string
}

// NewSandbox returns a sandbox with workspace-only filesystem by default.
func NewSandbox() *Sandbox {
	wd, _ := os.Getwd()
	return &Sandbox{
		Network:    false,
		Filesystem: FsWorkspaceOnly,
		WorkDir:    wd,
	}
}

// NewSandboxFromConfig builds a sandbox from configuration values.
func NewSandboxFromConfig(enabled bool, network bool, fsMode string, workDir string, allowPaths []string) *Sandbox {
	s := &Sandbox{
		Network:    network,
		WorkDir:    workDir,
		AllowPaths: allowPaths,
	}
	if !enabled {
		s.Filesystem = FsOff
	} else {
		switch FilesystemIsolationMode(fsMode) {
		case FsOff, FsWorkspaceOnly, FsAllowList:
			s.Filesystem = FilesystemIsolationMode(fsMode)
		default:
			s.Filesystem = FsWorkspaceOnly
		}
	}
	return s
}

// Run executes fn within the sandbox. For now Run is a pass-through;
// filesystem validation is done separately via ValidatePath.
func (s *Sandbox) Run(ctx context.Context, fn func() error) error {
	return fn()
}

// ValidatePath checks whether a path is accessible under the current sandbox policy.
// Returns the canonical path and nil if allowed, or an error if denied.
func (s *Sandbox) ValidatePath(target string) (string, error) {
	if s.Filesystem == FsOff {
		return filepath.Abs(target)
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If the file does not exist yet, trust abs (writes are typically to new files).
		if os.IsNotExist(err) {
			real = abs
		} else {
			return "", err
		}
	}

	switch s.Filesystem {
	case FsWorkspaceOnly:
		if !strings.HasPrefix(real, s.WorkDir+string(os.PathSeparator)) && real != s.WorkDir {
			return "", &SandboxError{Path: target, Reason: "outside workspace"}
		}

	case FsAllowList:
		allowed := false
		for _, p := range s.AllowPaths {
			ap, err := filepath.Abs(p)
			if err != nil {
				continue
			}
			if strings.HasPrefix(real, ap+string(os.PathSeparator)) || real == ap {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", &SandboxError{Path: target, Reason: "not in allow list"}
		}
	}

	return real, nil
}

// SandboxError describes a path that was denied by sandbox policy.
type SandboxError struct {
	Path   string
	Reason string
}

func (e *SandboxError) Error() string {
	return "sandbox: " + e.Reason + ": " + e.Path
}
