package jobtech

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Database interface to support pgxpool and mocks
type Database interface {
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
}

// Repository handles database operations for the JobTechnology model.
type Repository struct {
	db Database
}

// NewRepository creates a new Repository instance.
func NewRepository(db Database) *Repository {
	return &Repository{db: db}
}

// Create inserts a new job-technology association into the database.
func (r *Repository) Create(ctx context.Context, jobTech *JobTechnology) error {
	err := r.db.QueryRow(
		ctx,
		createJobTechnologyQuery,
		jobTech.JobID,
		jobTech.TechnologyID,
		jobTech.IsRequired,
	).Scan(&jobTech.ID, &jobTech.CreatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate job-technology association)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &DuplicateError{
				JobID:        jobTech.JobID,
				TechnologyID: jobTech.TechnologyID,
			}
		}
		return fmt.Errorf("failed to create job technology association: %w", err)
	}

	return nil
}

// GetByJobAndTechnology retrieves a job-technology association by job ID and technology ID.
func (r *Repository) GetByJobAndTechnology(ctx context.Context, jobID, technologyID int) (*JobTechnology, error) {
	jobTech := &JobTechnology{}
	err := r.db.QueryRow(ctx, getJobTechnologyByJobAndTechQuery, jobID, technologyID).Scan(
		&jobTech.ID,
		&jobTech.JobID,
		&jobTech.TechnologyID,
		&jobTech.IsRequired,
		&jobTech.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &NotFoundError{
				JobID:        jobID,
				TechnologyID: technologyID,
			}
		}
		return nil, fmt.Errorf("failed to get job technology association: %w", err)
	}

	return jobTech, nil
}

// Update updates an existing job-technology association in the database.
func (r *Repository) Update(ctx context.Context, jobTech *JobTechnology) error {
	commandTag, err := r.db.Exec(
		ctx,
		updateJobTechnologyQuery,
		jobTech.IsRequired,
		jobTech.ID,
	)

	if err != nil {
		// Check for unique constraint violation (duplicate job-technology association)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &DuplicateError{
				JobID:        jobTech.JobID,
				TechnologyID: jobTech.TechnologyID,
			}
		}
		return fmt.Errorf("failed to update job technology association: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return &NotFoundError{ID: jobTech.ID}
	}

	return nil
}

// Delete removes a job-technology association from the database.
func (r *Repository) Delete(ctx context.Context, id int) error {
	commandTag, err := r.db.Exec(ctx, deleteJobTechnologyQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete job technology association: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return &NotFoundError{ID: id}
	}

	return nil
}

// ListByJob retrieves all technology associations for a specific job.
func (r *Repository) ListByJob(ctx context.Context, jobID int) ([]*JobTechnology, error) {
	rows, err := r.db.Query(ctx, listJobTechnologiesByJobQuery, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list job technologies: %w", err)
	}
	defer rows.Close()

	var jobTechnologies []*JobTechnology
	for rows.Next() {
		jobTech := &JobTechnology{}
		err = rows.Scan(
			&jobTech.ID,
			&jobTech.JobID,
			&jobTech.TechnologyID,
			&jobTech.IsRequired,
			&jobTech.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job technology row: %w", err)
		}
		jobTechnologies = append(jobTechnologies, jobTech)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job technology rows: %w", err)
	}

	return jobTechnologies, nil
}

// ListByTechnology retrieves all job associations for a specific technology.
func (r *Repository) ListByTechnology(ctx context.Context, technologyID int) ([]*JobTechnology, error) {
	rows, err := r.db.Query(ctx, listJobTechnologiesByTechnologyQuery, technologyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list technology jobs: %w", err)
	}
	defer rows.Close()

	var jobTechnologies []*JobTechnology
	for rows.Next() {
		jobTech := &JobTechnology{}
		err = rows.Scan(
			&jobTech.ID,
			&jobTech.JobID,
			&jobTech.TechnologyID,
			&jobTech.IsRequired,
			&jobTech.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan technology job row: %w", err)
		}
		jobTechnologies = append(jobTechnologies, jobTech)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating technology job rows: %w", err)
	}

	return jobTechnologies, nil
}

// GetJobTechnologiesBatch fetches technologies for multiple jobs in a single query
func (r *Repository) GetJobTechnologiesBatch(ctx context.Context, jobIDs []int) (
	map[int][]*JobTechnologyWithDetails, error) {
	if len(jobIDs) == 0 {
		return make(map[int][]*JobTechnologyWithDetails), nil
	}

	// Build query with IN clause for multiple job IDs
	placeholders := make([]string, len(jobIDs))
	args := make([]any, len(jobIDs))
	for i, jobID := range jobIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = jobID
	}

	query := fmt.Sprintf(getJobTechnologiesBatchQuery, strings.Join(placeholders, ","))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get job technologies: %w", err)
	}
	defer rows.Close()

	// Group technologies by job ID
	technologiesMap := make(map[int][]*JobTechnologyWithDetails)
	for rows.Next() {
		tech := &JobTechnologyWithDetails{}
		err = rows.Scan(
			&tech.JobID,
			&tech.TechnologyID,
			&tech.IsRequired,
			&tech.TechName,
			&tech.TechCategory,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job technology row: %w", err)
		}
		technologiesMap[tech.JobID] = append(technologiesMap[tech.JobID], tech)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job technology rows: %w", err)
	}

	return technologiesMap, nil
}
