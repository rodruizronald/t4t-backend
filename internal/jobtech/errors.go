// Package jobtech provides functionality for managing company entities
// including CRUD operations, error handling, and business logic.
package jobtech

import (
	"errors"
	"fmt"
)

// NotFoundError represents a job technology association not found error
type NotFoundError struct {
	ID           int
	JobID        int
	TechnologyID int
}

func (e NotFoundError) Error() string {
	if e.ID > 0 {
		return fmt.Sprintf("job technology with ID %d not found", e.ID)
	}
	return fmt.Sprintf("job technology association for job %d and technology %d not found", e.JobID, e.TechnologyID)
}

// IsNotFound checks if an error is a job technology not found error
func IsNotFound(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// DuplicateError represents a duplicate job technology association error
type DuplicateError struct {
	JobID        int
	TechnologyID int
}

func (e DuplicateError) Error() string {
	return fmt.Sprintf("job technology association for job %d and technology %d already exists", e.JobID, e.TechnologyID)
}

// IsDuplicate checks if an error is a duplicate job technology error
func IsDuplicate(err error) bool {
	var duplicateErr *DuplicateError
	return errors.As(err, &duplicateErr)
}
