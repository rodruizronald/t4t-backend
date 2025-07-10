// Package techalias provides functionality for managing company entities
// including CRUD operations, error handling, and business logic.
package techalias

import (
	"errors"
	"fmt"
)

// NotFoundError represents a technology alias not found error
type NotFoundError struct {
	ID    int
	Alias string
}

func (e NotFoundError) Error() string {
	if e.ID != 0 {
		return fmt.Sprintf("technology alias with ID %d not found", e.ID)
	}
	return fmt.Sprintf("technology alias with value %q not found", e.Alias)
}

// IsNotFound checks if an error is a technology alias not found error
func IsNotFound(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// DuplicateError represents a duplicate technology alias error
type DuplicateError struct {
	Alias string
}

func (e DuplicateError) Error() string {
	return fmt.Sprintf("technology alias %q already exists", e.Alias)
}

// IsDuplicate checks if an error is a duplicate technology alias error
func IsDuplicate(err error) bool {
	var duplicateErr *DuplicateError
	return errors.As(err, &duplicateErr)
}
