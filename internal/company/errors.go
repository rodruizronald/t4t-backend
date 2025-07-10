// Package company provides functionality for managing company entities
// including CRUD operations, error handling, and business logic.
package company

import (
	"errors"
	"fmt"
)

// NotFoundError represents a company not found error
type NotFoundError struct {
	ID   int
	Name string
}

func (e NotFoundError) Error() string {
	if e.ID > 0 {
		return fmt.Sprintf("company with ID %d not found", e.ID)
	}
	return fmt.Sprintf("company with name %s not found", e.Name)
}

// IsNotFound checks if an error is a company not found error
func IsNotFound(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// DuplicateError represents a duplicate company error
type DuplicateError struct {
	Name string
}

func (e DuplicateError) Error() string {
	return fmt.Sprintf("company with name %s already exists", e.Name)
}

// IsDuplicate checks if an error is a duplicate company error
func IsDuplicate(err error) bool {
	var duplicateErr *DuplicateError
	return errors.As(err, &duplicateErr)
}
