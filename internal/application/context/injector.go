// Package context provides application-level context management services.
package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	domainContext "github.com/jbctechsolutions/skillrunner/internal/domain/context"
)

// HeadlineContext represents a compact context summary for injection into prompts.
type HeadlineContext struct {
	Focus       string
	ActiveRules []string
	RecentItems []string
	TokenCount  int
}

// Injector generates headline context for workspace sessions.
type Injector struct {
	workspaceRepo ports.WorkspaceStateStoragePort
	ruleRepo      ports.RuleStoragePort
	itemRepo      ports.ContextItemStoragePort
	estimator     *Estimator
	maxTokens     int
}

// NewInjector creates a new headline context injector.
func NewInjector(
	workspaceRepo ports.WorkspaceStateStoragePort,
	ruleRepo ports.RuleStoragePort,
	itemRepo ports.ContextItemStoragePort,
) *Injector {
	return &Injector{
		workspaceRepo: workspaceRepo,
		ruleRepo:      ruleRepo,
		itemRepo:      itemRepo,
		estimator:     NewEstimator(),
		maxTokens:     500, // Default budget
	}
}

// SetMaxTokens configures the maximum token budget for headline context.
func (i *Injector) SetMaxTokens(max int) {
	if max > 0 {
		i.maxTokens = max
	}
}

// Generate builds a headline context for the given workspace.
// Stays under the configured token budget by prioritizing:
// 1. Current focus
// 2. Active global rules
// 3. Active workspace rules
// 4. Recently used context items
func (i *Injector) Generate(ctx context.Context, workspaceID string) (*HeadlineContext, error) {
	workspace, err := i.workspaceRepo.Get(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	headline := &HeadlineContext{
		ActiveRules: make([]string, 0),
		RecentItems: make([]string, 0),
	}

	var parts []string
	remainingTokens := i.maxTokens

	// 1. Add focus if set (highest priority)
	if workspace.Focus() != "" {
		focusText := fmt.Sprintf("Current Focus: %s", workspace.Focus())
		tokens := i.estimator.Estimate(focusText)
		if tokens <= remainingTokens {
			headline.Focus = workspace.Focus()
			parts = append(parts, focusText)
			remainingTokens -= tokens
		}
	}

	// 2. Add active global rules
	if remainingTokens > 50 { // Reserve some space
		globalRules, err := i.loadRulesByScope(ctx, domainContext.RuleScopeGlobal, remainingTokens/2)
		if err == nil {
			for _, rule := range globalRules {
				tokens := i.estimator.Estimate(rule)
				if tokens <= remainingTokens {
					headline.ActiveRules = append(headline.ActiveRules, rule)
					parts = append(parts, rule)
					remainingTokens -= tokens
				}
			}
		}
	}

	// 3. Add active workspace rules
	if remainingTokens > 50 {
		workspaceRules, err := i.loadRulesByScope(ctx, domainContext.RuleScopeWorkspace, remainingTokens/2)
		if err == nil {
			for _, rule := range workspaceRules {
				tokens := i.estimator.Estimate(rule)
				if tokens <= remainingTokens {
					headline.ActiveRules = append(headline.ActiveRules, rule)
					parts = append(parts, rule)
					remainingTokens -= tokens
				}
			}
		}
	}

	// 4. Add recently used context items if there's space
	if remainingTokens > 30 {
		items, err := i.itemRepo.List(ctx)
		if err == nil && len(items) > 0 {
			// Sort by last used (most recent first)
			// For now, just take first few items
			for _, item := range items {
				if remainingTokens <= 30 {
					break
				}

				itemText := fmt.Sprintf("Item: %s", item.Name())
				tokens := i.estimator.Estimate(itemText)
				if tokens <= remainingTokens {
					headline.RecentItems = append(headline.RecentItems, item.Name())
					parts = append(parts, itemText)
					remainingTokens -= tokens
				}
			}
		}
	}

	// Calculate actual token count
	fullText := strings.Join(parts, "\n")
	headline.TokenCount = i.estimator.Estimate(fullText)

	return headline, nil
}

// Format returns a formatted string representation of the headline context.
func (h *HeadlineContext) Format() string {
	var parts []string

	if h.Focus != "" {
		parts = append(parts, fmt.Sprintf("Focus: %s", h.Focus))
	}

	if len(h.ActiveRules) > 0 {
		parts = append(parts, "\nActive Rules:")
		for _, rule := range h.ActiveRules {
			parts = append(parts, fmt.Sprintf("- %s", rule))
		}
	}

	if len(h.RecentItems) > 0 {
		parts = append(parts, "\nRecent Context:")
		for _, item := range h.RecentItems {
			parts = append(parts, fmt.Sprintf("- %s", item))
		}
	}

	return strings.Join(parts, "\n")
}

// loadRulesByScope loads active rules for a specific scope within a token budget.
func (i *Injector) loadRulesByScope(ctx context.Context, scope domainContext.RuleScope, budget int) ([]string, error) {
	rules, err := i.ruleRepo.ListByScope(ctx, scope)
	if err != nil {
		return nil, err
	}

	var result []string
	remainingBudget := budget

	for _, rule := range rules {
		if !rule.IsActive() {
			continue
		}

		// Format rule
		ruleText := fmt.Sprintf("%s: %s", rule.Name(), rule.Content())

		// Estimate tokens
		tokens := i.estimator.Estimate(ruleText)

		// Truncate if needed
		if tokens > remainingBudget {
			// Try to fit a truncated version
			if remainingBudget > 20 {
				truncated, _ := i.estimator.TruncateToFit(ruleText, remainingBudget)
				result = append(result, truncated+"...")
				break
			}
			break
		}

		result = append(result, ruleText)
		remainingBudget -= tokens

		if remainingBudget <= 0 {
			break
		}
	}

	return result, nil
}
