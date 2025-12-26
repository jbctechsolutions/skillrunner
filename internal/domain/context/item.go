// Package context provides domain entities for workspace and context management.
package context

import (
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// ItemType represents the type of context item.
type ItemType string

const (
	// ItemTypeFile represents a file reference.
	ItemTypeFile ItemType = "file"

	// ItemTypeSnippet represents a code or text snippet.
	ItemTypeSnippet ItemType = "snippet"

	// ItemTypeURL represents a URL reference.
	ItemTypeURL ItemType = "url"
)

// ContextItem represents a piece of context that can be loaded into a skill session.
// Items can be files, code snippets, or URLs that provide relevant information.
type ContextItem struct {
	id            string
	name          string
	itemType      ItemType
	content       string
	tags          []string
	tokenEstimate int
	lastUsedAt    time.Time
	createdAt     time.Time
}

// NewContextItem creates a new ContextItem with the required fields.
// Returns an error if validation fails:
//   - id is required
//   - name is required
//   - itemType must be valid
func NewContextItem(id, name string, itemType ItemType) (*ContextItem, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)

	if id == "" {
		return nil, errors.New("context_item", "context item ID is required")
	}
	if name == "" {
		return nil, errors.New("context_item", "context item name is required")
	}

	// Validate item type
	switch itemType {
	case ItemTypeFile, ItemTypeSnippet, ItemTypeURL:
		// Valid type
	default:
		return nil, errors.New("context_item", "invalid item type")
	}

	now := time.Now()
	return &ContextItem{
		id:         id,
		name:       name,
		itemType:   itemType,
		tags:       make([]string, 0),
		createdAt:  now,
		lastUsedAt: now,
	}, nil
}

// ID returns the item's unique identifier.
func (i *ContextItem) ID() string {
	return i.id
}

// Name returns the item's name.
func (i *ContextItem) Name() string {
	return i.name
}

// Type returns the item type.
func (i *ContextItem) Type() ItemType {
	return i.itemType
}

// Content returns the item's content.
func (i *ContextItem) Content() string {
	return i.content
}

// Tags returns a copy of the item's tags.
func (i *ContextItem) Tags() []string {
	tags := make([]string, len(i.tags))
	copy(tags, i.tags)
	return tags
}

// TokenEstimate returns the estimated token count for this item.
func (i *ContextItem) TokenEstimate() int {
	return i.tokenEstimate
}

// LastUsedAt returns when the item was last used.
func (i *ContextItem) LastUsedAt() time.Time {
	return i.lastUsedAt
}

// CreatedAt returns when the item was created.
func (i *ContextItem) CreatedAt() time.Time {
	return i.createdAt
}

// SetContent sets the item's content.
func (i *ContextItem) SetContent(content string) {
	i.content = content
}

// SetTokenEstimate sets the estimated token count.
func (i *ContextItem) SetTokenEstimate(tokens int) {
	if tokens < 0 {
		tokens = 0
	}
	i.tokenEstimate = tokens
}

// AddTag adds a tag to the item.
func (i *ContextItem) AddTag(tag string) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return
	}

	// Avoid duplicates
	for _, t := range i.tags {
		if t == tag {
			return
		}
	}

	i.tags = append(i.tags, tag)
}

// SetTags replaces the item's tags.
func (i *ContextItem) SetTags(tags []string) {
	i.tags = make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			i.tags = append(i.tags, tag)
		}
	}
}

// RemoveTag removes a tag from the item.
func (i *ContextItem) RemoveTag(tag string) {
	tag = strings.TrimSpace(tag)
	for idx, t := range i.tags {
		if t == tag {
			i.tags = append(i.tags[:idx], i.tags[idx+1:]...)
			return
		}
	}
}

// HasTag checks if the item has a specific tag.
func (i *ContextItem) HasTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	for _, t := range i.tags {
		if t == tag {
			return true
		}
	}
	return false
}

// MarkUsed updates the last used timestamp.
func (i *ContextItem) MarkUsed() {
	i.lastUsedAt = time.Now()
}

// Validate checks if the ContextItem is in a valid state.
func (i *ContextItem) Validate() error {
	if strings.TrimSpace(i.id) == "" {
		return errors.New("context_item", "context item ID is required")
	}
	if strings.TrimSpace(i.name) == "" {
		return errors.New("context_item", "context item name is required")
	}

	// Validate item type
	switch i.itemType {
	case ItemTypeFile, ItemTypeSnippet, ItemTypeURL:
		// Valid type
	default:
		return errors.New("context_item", "invalid item type")
	}

	return nil
}
