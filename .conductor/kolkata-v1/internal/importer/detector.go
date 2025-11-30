package importer

import (
	"os"
	"strings"
)

// SourceType represents the type of skill source
type SourceType int

const (
	SourceTypeUnknown   SourceType = iota
	SourceTypeWebHTTP              // https://example.com/skill
	SourceTypeLocalPath            // /path/to/skill or ~/skill
	SourceTypeGitSSH               // git@github.com:user/repo.git
	SourceTypeGitHTTPS             // https://github.com/user/repo.git
)

// String returns the string representation of SourceType
func (st SourceType) String() string {
	switch st {
	case SourceTypeWebHTTP:
		return "web-http"
	case SourceTypeLocalPath:
		return "local-path"
	case SourceTypeGitSSH:
		return "git-ssh"
	case SourceTypeGitHTTPS:
		return "git-https"
	default:
		return "unknown"
	}
}

// DetectSourceType determines the type of source from a string
func DetectSourceType(source string) SourceType {
	// Check for HTTP/HTTPS URLs
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Check for raw file URLs (GitHub, GitLab, etc.)
		if strings.Contains(source, "/raw/") ||
			strings.Contains(source, "raw.githubusercontent.com") ||
			strings.Contains(source, "raw.gitlab.com") {
			return SourceTypeWebHTTP
		}

		// Check if it's a git repo URL
		if strings.HasSuffix(source, ".git") {
			return SourceTypeGitHTTPS
		}

		// Check for common git hosting patterns
		if strings.Contains(source, "github.com") &&
			(strings.Contains(source, "/tree/") || strings.Contains(source, "/blob/")) {
			// This is a GitHub web UI URL, not a clonable repo
			return SourceTypeWebHTTP
		}

		// Simple GitHub/GitLab repo URLs without .git
		// e.g., https://github.com/user/repo
		if (strings.Contains(source, "github.com") || strings.Contains(source, "gitlab.com")) &&
			len(strings.Split(strings.TrimPrefix(strings.TrimPrefix(source, "https://"), "http://"), "/")) <= 3 {
			return SourceTypeGitHTTPS
		}

		return SourceTypeWebHTTP
	}

	// Check for git SSH URLs
	if strings.HasPrefix(source, "git@") {
		return SourceTypeGitSSH
	}

	// Check for .git suffix (could be relative path to git repo)
	if strings.HasSuffix(source, ".git") {
		return SourceTypeGitHTTPS
	}

	// Check for local path indicators
	if strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") ||
		strings.HasPrefix(source, ".") || strings.HasPrefix(source, "..") {
		return SourceTypeLocalPath
	}

	// Try to stat as a relative path
	if _, err := os.Stat(source); err == nil {
		return SourceTypeLocalPath
	}

	return SourceTypeUnknown
}

// NormalizeSource normalizes a source string for processing
func NormalizeSource(source string) string {
	// Expand ~ to home directory
	if strings.HasPrefix(source, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			source = strings.Replace(source, "~", home, 1)
		}
	}

	// Trim whitespace
	source = strings.TrimSpace(source)

	return source
}
