package jobtech

import (
	"time"
)

// JobTechnology represents the association between a job and a technology,
// including additional metadata about the relationship.
type JobTechnology struct {
	ID           int       `db:"id"`
	JobID        int       `db:"job_id"`
	TechnologyID int       `db:"technology_id"`
	IsRequired   bool      `db:"is_required"`
	CreatedAt    time.Time `db:"created_at"`
}

// JobTechnologyWithDetails represents a job-technology association with full technology details
type JobTechnologyWithDetails struct {
	JobID        int    `db:"job_id"`
	TechnologyID int    `db:"technology_id"`
	TechName     string `db:"tech_name"`
	TechCategory string `db:"tech_category"`
	IsRequired   bool   `db:"is_required"`
}
