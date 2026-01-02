// Package context provides domain entities for workspace and context management.
package context

import (
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
)

// RuleScope represents the scope of a rule.
type RuleScope string

const (
	// RuleScopeGlobal indicates the rule applies globally.
	RuleScopeGlobal RuleScope = "global"

	// RuleScopeWorkspace indicates the rule applies to a specific workspace.
	RuleScopeWorkspace RuleScope = "workspace"

	// RuleScopeSession indicates the rule applies to a specific session.
	RuleScopeSession RuleScope = "session"
)

// Rule represents a context rule or guideline that applies during skill execution.
// Rules can be global, workspace-specific, or session-specific.
type Rule struct {
	id        string
	name      string
	content   string
	scope     RuleScope
	isActive  bool
	createdAt time.Time
}

// NewRule creates a new Rule with the required fields.
// Returns an error if validation fails:
//   - id is required
//   - name is required
//   - content is required
//   - scope must be valid
func NewRule(id, name, content string, scope RuleScope) (*Rule, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	content = strings.TrimSpace(content)

	if id == "" {
		return nil, errors.New("rule", "rule ID is required")
	}
	if name == "" {
		return nil, errors.New("rule", "rule name is required")
	}
	if content == "" {
		return nil, errors.New("rule", "rule content is required")
	}

	// Validate scope
	switch scope {
	case RuleScopeGlobal, RuleScopeWorkspace, RuleScopeSession:
		// Valid scope
	default:
		return nil, errors.New("rule", "invalid rule scope")
	}

	return &Rule{
		id:        id,
		name:      name,
		content:   content,
		scope:     scope,
		isActive:  true,
		createdAt: time.Now(),
	}, nil
}

// ID returns the rule's unique identifier.
func (r *Rule) ID() string {
	return r.id
}

// Name returns the rule's name.
func (r *Rule) Name() string {
	return r.name
}

// Content returns the rule's content.
func (r *Rule) Content() string {
	return r.content
}

// Scope returns the rule's scope.
func (r *Rule) Scope() RuleScope {
	return r.scope
}

// IsActive returns whether the rule is currently active.
func (r *Rule) IsActive() bool {
	return r.isActive
}

// CreatedAt returns when the rule was created.
func (r *Rule) CreatedAt() time.Time {
	return r.createdAt
}

// SetContent updates the rule's content.
func (r *Rule) SetContent(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("rule", "rule content is required")
	}
	r.content = content
	return nil
}

// Activate marks the rule as active.
func (r *Rule) Activate() {
	r.isActive = true
}

// Deactivate marks the rule as inactive.
func (r *Rule) Deactivate() {
	r.isActive = false
}

// Validate checks if the Rule is in a valid state.
func (r *Rule) Validate() error {
	if strings.TrimSpace(r.id) == "" {
		return errors.New("rule", "rule ID is required")
	}
	if strings.TrimSpace(r.name) == "" {
		return errors.New("rule", "rule name is required")
	}
	if strings.TrimSpace(r.content) == "" {
		return errors.New("rule", "rule content is required")
	}

	// Validate scope
	switch r.scope {
	case RuleScopeGlobal, RuleScopeWorkspace, RuleScopeSession:
		// Valid scope
	default:
		return errors.New("rule", "invalid rule scope")
	}

	return nil
}
