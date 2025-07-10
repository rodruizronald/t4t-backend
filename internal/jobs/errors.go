// Package jobs provides functionality for managing company entities
// including CRUD operations, error handling, and business logic.
package jobs

import (
	"errors"
	"fmt"
)

// NotFoundError represents a job not found error
type NotFoundError struct {
	ID        int
	Signature string
}

func (e NotFoundError) Error() string {
	if e.ID != 0 {
		return fmt.Sprintf("job with ID %d not found", e.ID)
	}
	return fmt.Sprintf("job with signature %s not found", e.Signature)
}

// IsNotFound checks if an error is a job not found error
func IsNotFound(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// DuplicateError represents a duplicate job error
type DuplicateError struct {
	Signature string
}

func (e DuplicateError) Error() string {
	return fmt.Sprintf("job with signature %s already exists", e.Signature)
}

// IsDuplicate checks if an error is a duplicate job error
func IsDuplicate(err error) bool {
	var duplicateErr *DuplicateError
	return errors.As(err, &duplicateErr)
}
