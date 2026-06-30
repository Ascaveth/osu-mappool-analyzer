package domain

import (
	"fmt"
	"sort"
)

// ConfigurationIssue is one problem found by ValidateConfiguration. Hard
// rejects (IsError) block the configuration from being accepted; warnings
// flag something that is almost always a mistake but isn't formally
// invalid (docs/07-tournament-configuration.md "Validation rules").
type ConfigurationIssue struct {
	IsError bool
	Message string
}

// ValidateConfiguration checks a Tournament against the cross-field rules
// docs/07-tournament-configuration.md defers to "application code" (rules
// that a JSON Schema shape check alone can't express): unique Category.Order
// within a Stage, Category slot count >= 1, and duplicate Category.Name
// within a Stage as a warning. It does not duplicate the JSON Schema's own
// shape checks (required fields, types) — those are enforced at the API
// boundary (docs/14-api-specification.md), not here.
//
// Stage.Order is deliberately NOT required to be unique: docs/07's
// "Supporting future/non-linear formats" section names same-Order stages
// as the explicit mechanism for parallel/concurrent stages (e.g.
// simultaneous group pools), to be treated as a peer set rather than an
// ValidateConfiguration checks a tournament for category configuration issues.
// It reports hard errors for categories with no slots and for duplicate category
// orders within a stage, and reports warnings when a category name is reused
// within a stage.
//
// @returns A slice of configuration issues found while validating the tournament.
func ValidateConfiguration(t *Tournament) []ConfigurationIssue {
	var issues []ConfigurationIssue

	if t == nil {
		return []ConfigurationIssue{{IsError: true, Message: "tournament configuration is nil"}}
	}

	for _, stage := range t.Stages {
		seenCategoryOrders := map[int][]string{}
		seenCategoryNames := map[string]int{}
		for _, category := range stage.Categories {
			seenCategoryOrders[category.Order] = append(seenCategoryOrders[category.Order], category.Name)
			seenCategoryNames[category.Name]++

			if len(category.Slots) < 1 {
				issues = append(issues, ConfigurationIssue{
					IsError: true,
					Message: fmt.Sprintf("stage %q category %q has zero slots; a category with no slots should be omitted", stage.Name, category.Name),
				})
			}
		}

		orders := make([]int, 0, len(seenCategoryOrders))
		for order := range seenCategoryOrders {
			orders = append(orders, order)
		}
		sort.Ints(orders)
		for _, order := range orders {
			if names := seenCategoryOrders[order]; len(names) > 1 {
				issues = append(issues, ConfigurationIssue{
					IsError: true,
					Message: fmt.Sprintf("stage %q: category order %d is used by more than one category: %v", stage.Name, order, names),
				})
			}
		}

		names := make([]string, 0, len(seenCategoryNames))
		for name := range seenCategoryNames {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if count := seenCategoryNames[name]; count > 1 {
				issues = append(issues, ConfigurationIssue{
					IsError: false,
					Message: fmt.Sprintf("stage %q: category name %q is used %d times; this is usually a mistake", stage.Name, name, count),
				})
			}
		}
	}

	return issues
}

// HasErrors reports whether issues contains at least one hard rejection.
func HasErrors(issues []ConfigurationIssue) bool {
	for _, i := range issues {
		if i.IsError {
			return true
		}
	}
	return false
}
