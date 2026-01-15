// Package memory provides domain models for memory/context persistence across sessions.
package memory

import "errors"

// Domain-specific errors for memory operations.
// Note: The infrastructure Loader handles these conditions gracefully (skipping missing
// files, preventing cycles via deduplication, truncating on limit exceeded) rather than
// returning errors. These are defined for use by callers who need stricter error handling.
var (
	// ErrMemoryEmpty indicates that the memory content is empty.
	ErrMemoryEmpty = errors.New("memory content is empty")

	// ErrIncludeNotFound indicates that an included file could not be found.
	ErrIncludeNotFound = errors.New("included file not found")

	// ErrIncludeCycle indicates a circular dependency in include directives.
	ErrIncludeCycle = errors.New("circular include detected")

	// ErrTokenLimitExceeded indicates the memory content exceeds the configured token limit.
	ErrTokenLimitExceeded = errors.New("memory exceeds token limit")
)
