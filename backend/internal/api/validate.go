package api

import "fmt"

// validateTournamentConfiguration checks the wire-level shape constraints
// from docs/api/openapi.yaml's TournamentConfiguration/StageConfiguration/
// CategoryConfiguration schemas (minLength, minItems, minimum) that a plain
// JSON decode doesn't enforce. Cross-field domain rules (duplicate category
// order, etc.) are checked separately by domain.ValidateConfiguration once
// this passes.
func validateTournamentConfiguration(dto tournamentConfigurationDTO) []FieldError {
	var errs []FieldError

	if dto.Name == "" {
		errs = append(errs, FieldError{Field: "name", Message: "must not be empty"})
	}
	if len(dto.Stages) < 1 {
		errs = append(errs, FieldError{Field: "stages", Message: "must contain at least 1 item"})
	}

	for i, stage := range dto.Stages {
		prefix := fmt.Sprintf("stages[%d]", i)
		if stage.Name == "" {
			errs = append(errs, FieldError{Field: prefix + ".name", Message: "must not be empty"})
		}
		if stage.Order < 1 {
			errs = append(errs, FieldError{Field: prefix + ".order", Message: "must be >= 1"})
		}
		if len(stage.Categories) < 1 {
			errs = append(errs, FieldError{Field: prefix + ".categories", Message: "must contain at least 1 item"})
		}
		for j, cat := range stage.Categories {
			catPrefix := fmt.Sprintf("%s.categories[%d]", prefix, j)
			if cat.Name == "" {
				errs = append(errs, FieldError{Field: catPrefix + ".name", Message: "must not be empty"})
			}
			if cat.Order < 1 {
				errs = append(errs, FieldError{Field: catPrefix + ".order", Message: "must be >= 1"})
			}
			if cat.SlotCount < 1 {
				errs = append(errs, FieldError{Field: catPrefix + ".slotCount", Message: "must be >= 1"})
			}
		}
	}

	return errs
}
