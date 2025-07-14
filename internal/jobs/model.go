package jobs

import (
	"time"

	"github.com/lib/pq"
)

// Database entities and repository-level structs for job management.
// This file contains the core database models and search parameters used by the repository layer.
// These models map directly to database tables and are used for data persistence operations.

// Job represents the database entity
type Job struct {
	ID               int            `db:"id"`
	CompanyID        int            `db:"company_id"`
	Title            string         `db:"title"`
	OriginalPost     string         `db:"original_post"`
	Description      string         `db:"description"`
	Responsibilities pq.StringArray `db:"responsibilities"`
	SkillMustHave    pq.StringArray `db:"skill_must_have"`
	SkillNiceToHave  pq.StringArray `db:"skill_nice_to_have"`
	Benefits         pq.StringArray `db:"benefits"`
	ExperienceLevel  string         `db:"experience_level"`
	EmploymentType   string         `db:"employment_type"`
	Location         string         `db:"location"`
	WorkMode         string         `db:"work_mode"`
	ApplicationURL   string         `db:"application_url"`
	IsActive         bool           `db:"is_active"`
	Signature        string         `db:"signature"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

// JobWithCompany represents a job with company details (for read operations only)
type JobWithCompany struct {
	Job                   // Embed the original Job struct
	CompanyName    string `db:"company_name"`
	CompanyLogoURL string `db:"company_logo_url"`
}

// SearchParams defines parameters for job search (repository layer)
type SearchParams struct {
	Query           string
	Limit           int
	Offset          int
	ExperienceLevel *string
	EmploymentType  *string
	Location        *string
	WorkMode        *string
	Company         *string
	DateFrom        *time.Time
	DateTo          *time.Time
}

// GetLimit returns the limit for pagination to satisfy httpservice.SearchParams interface
func (sp *SearchParams) GetLimit() int {
	return sp.Limit
}

// GetOffset returns the offset for pagination to satisfy httpservice.SearchParams interface
func (sp *SearchParams) GetOffset() int {
	return sp.Offset
}
