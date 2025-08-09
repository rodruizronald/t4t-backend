package jobs

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Create(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		job          *Job
		mockSetup    func(mock pgxmock.PgxPoolIface, job *Job)
		checkResults func(t *testing.T, result *Job, err error)
	}{
		{
			name: "successful creation",
			job: &Job{
				CompanyID:       1,
				Title:           "Software Engineer",
				Description:     "Job description",
				ExperienceLevel: "Mid-Level",
				EmploymentType:  "Full-Time",
				Location:        "San Francisco",
				WorkMode:        "Remote",
				ApplicationURL:  "https://example.com/apply",
				IsActive:        true,
				Signature:       "job-signature-1",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, job *Job) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createJobQuery)).
					WithArgs(
						job.CompanyID,
						job.Title,
						job.Description,
						job.ExperienceLevel,
						job.EmploymentType,
						job.Location,
						job.WorkMode,
						job.ApplicationURL,
						job.IsActive,
						job.Signature,
					).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "created_at", "updated_at",
					}).AddRow(1, now, now))
			},
			checkResults: func(t *testing.T, result *Job, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, now, result.CreatedAt)
				assert.Equal(t, now, result.UpdatedAt)
			},
		},
		{
			name: "duplicate job signature",
			job: &Job{
				CompanyID:       2,
				Title:           "Product Manager",
				Description:     "Job description",
				ExperienceLevel: "Senior",
				EmploymentType:  "Full-Time",
				Location:        "New York",
				WorkMode:        "Hybrid",
				ApplicationURL:  "https://example.com/apply2",
				IsActive:        true,
				Signature:       "duplicate-signature",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, job *Job) {
				t.Helper()
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "idx_jobs_signature",
				}
				mock.ExpectQuery(regexp.QuoteMeta(createJobQuery)).
					WithArgs(
						job.CompanyID,
						job.Title,
						job.Description,
						job.ExperienceLevel,
						job.EmploymentType,
						job.Location,
						job.WorkMode,
						job.ApplicationURL,
						job.IsActive,
						job.Signature,
					).
					WillReturnError(pgErr)
			},
			checkResults: func(t *testing.T, _ *Job, err error) {
				t.Helper()
				require.Error(t, err)

				var duplicateErr *DuplicateError
				require.ErrorAs(t, err, &duplicateErr)
				assert.Equal(t, "duplicate-signature", duplicateErr.Signature)
			},
		},
		{
			name: "database error",
			job: &Job{
				CompanyID:       3,
				Title:           "Data Scientist",
				Description:     "Job description",
				ExperienceLevel: "Entry-Level",
				EmploymentType:  "Contract",
				Location:        "Chicago",
				WorkMode:        "On-Site",
				ApplicationURL:  "https://example.com/apply3",
				IsActive:        true,
				Signature:       "job-signature-3",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, job *Job) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createJobQuery)).
					WithArgs(
						job.CompanyID,
						job.Title,
						job.Description,
						job.ExperienceLevel,
						job.EmploymentType,
						job.Location,
						job.WorkMode,
						job.ApplicationURL,
						job.IsActive,
						job.Signature,
					).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, _ *Job, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorIs(t, err, dbError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockDB.Close()

			repo := NewRepository(mockDB)
			tt.mockSetup(mockDB, tt.job)

			err = repo.Create(context.Background(), tt.job)
			tt.checkResults(t, tt.job, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetByID(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		jobID        int
		mockSetup    func(mock pgxmock.PgxPoolIface, jobID int)
		checkResults func(t *testing.T, result *Job, err error)
	}{
		{
			name:  "job found",
			jobID: 1,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobByIDQuery)).
					WithArgs(jobID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", "description", "experience_level", "employment_type",
						"location", "work_mode", "application_url", "is_active", "signature", "created_at", "updated_at",
					}).AddRow(
						1, 1, "Software Engineer", "Job description", "Mid-Level", "Full-Time",
						"San Francisco", "Remote", "https://example.com/apply", true, "job-signature-1", now, now,
					))
			},
			checkResults: func(t *testing.T, result *Job, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, 1, result.CompanyID)
				assert.Equal(t, "Software Engineer", result.Title)
				assert.Equal(t, "Job description", result.Description)
				assert.Equal(t, "Mid-Level", result.ExperienceLevel)
				assert.Equal(t, "Full-Time", result.EmploymentType)
				assert.Equal(t, "San Francisco", result.Location)
				assert.Equal(t, "Remote", result.WorkMode)
				assert.Equal(t, "https://example.com/apply", result.ApplicationURL)
				assert.True(t, result.IsActive)
				assert.Equal(t, "job-signature-1", result.Signature)
				assert.Equal(t, now, result.CreatedAt)
				assert.Equal(t, now, result.UpdatedAt)
			},
		},
		{
			name:  "job not found",
			jobID: 999,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobByIDQuery)).
					WithArgs(jobID).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *Job, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)

				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, 999, notFoundErr.ID)
			},
		},
		{
			name:  "database error",
			jobID: 2,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobByIDQuery)).
					WithArgs(jobID).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *Job, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
				require.ErrorIs(t, err, dbError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockDB.Close()

			repo := NewRepository(mockDB)
			tt.mockSetup(mockDB, tt.jobID)

			result, err := repo.GetByID(context.Background(), tt.jobID)
			tt.checkResults(t, result, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_Update(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		job          *Job
		mockSetup    func(mock pgxmock.PgxPoolIface, job *Job)
		checkResults func(t *testing.T, result *Job, err error)
	}{
		{
			name: "successful update",
			job: &Job{
				ID:              1,
				CompanyID:       1,
				Title:           "Updated Software Engineer",
				Description:     "Updated job description",
				ExperienceLevel: "Senior",
				EmploymentType:  "Full-Time",
				Location:        "San Francisco",
				WorkMode:        "Hybrid",
				ApplicationURL:  "https://example.com/apply-updated",
				IsActive:        true,
				Signature:       "job-signature-1-updated",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, job *Job) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(updateJobQuery)).
					WithArgs(
						job.CompanyID,
						job.Title,
						job.Description,
						job.ExperienceLevel,
						job.EmploymentType,
						job.Location,
						job.WorkMode,
						job.ApplicationURL,
						job.IsActive,
						job.Signature,
						job.ID,
					).
					WillReturnRows(pgxmock.NewRows([]string{"updated_at"}).AddRow(now))
			},
			checkResults: func(t *testing.T, result *Job, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, now, result.UpdatedAt)
			},
		},
		{
			name: "job not found",
			job: &Job{
				ID:              999,
				CompanyID:       1,
				Title:           "Nonexistent Job",
				Description:     "Job description",
				ExperienceLevel: "Mid-Level",
				EmploymentType:  "Full-Time",
				Location:        "Remote",
				WorkMode:        "Remote",
				ApplicationURL:  "https://example.com/apply",
				IsActive:        true,
				Signature:       "nonexistent-signature",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, job *Job) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(updateJobQuery)).
					WithArgs(
						job.CompanyID,
						job.Title,
						job.Description,
						job.ExperienceLevel,
						job.EmploymentType,
						job.Location,
						job.WorkMode,
						job.ApplicationURL,
						job.IsActive,
						job.Signature,
						job.ID,
					).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, _ *Job, err error) {
				t.Helper()
				require.Error(t, err)

				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, 999, notFoundErr.ID)
			},
		},
		{
			name: "duplicate job signature",
			job: &Job{
				ID:              2,
				CompanyID:       1,
				Title:           "Product Manager",
				Description:     "Job description",
				ExperienceLevel: "Senior",
				EmploymentType:  "Full-Time",
				Location:        "New York",
				WorkMode:        "Hybrid",
				ApplicationURL:  "https://example.com/apply2",
				IsActive:        true,
				Signature:       "duplicate-signature",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, job *Job) {
				t.Helper()
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "idx_jobs_signature",
				}
				mock.ExpectQuery(regexp.QuoteMeta(updateJobQuery)).
					WithArgs(
						job.CompanyID,
						job.Title,
						job.Description,
						job.ExperienceLevel,
						job.EmploymentType,
						job.Location,
						job.WorkMode,
						job.ApplicationURL,
						job.IsActive,
						job.Signature,
						job.ID,
					).
					WillReturnError(pgErr)
			},
			checkResults: func(t *testing.T, _ *Job, err error) {
				t.Helper()
				require.Error(t, err)

				var duplicateErr *DuplicateError
				require.ErrorAs(t, err, &duplicateErr)
				assert.Equal(t, "duplicate-signature", duplicateErr.Signature)
			},
		},
		{
			name: "database error",
			job: &Job{
				ID:              3,
				CompanyID:       1,
				Title:           "Error Job",
				Description:     "Job description",
				ExperienceLevel: "Mid-Level",
				EmploymentType:  "Full-Time",
				Location:        "Chicago",
				WorkMode:        "On-Site",
				ApplicationURL:  "https://example.com/apply3",
				IsActive:        true,
				Signature:       "error-signature",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, job *Job) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(updateJobQuery)).
					WithArgs(
						job.CompanyID,
						job.Title,
						job.Description,
						job.ExperienceLevel,
						job.EmploymentType,
						job.Location,
						job.WorkMode,
						job.ApplicationURL,
						job.IsActive,
						job.Signature,
						job.ID,
					).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, _ *Job, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorIs(t, err, dbError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockDB.Close()

			repo := NewRepository(mockDB)
			tt.mockSetup(mockDB, tt.job)

			err = repo.Update(context.Background(), tt.job)
			tt.checkResults(t, tt.job, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		id           int
		mockSetup    func(mock pgxmock.PgxPoolIface, id int)
		checkResults func(t *testing.T, err error)
	}{
		{
			name: "successful deletion",
			id:   1,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(deleteJobQuery)).
					WithArgs(id).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "job not found",
			id:   999,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(deleteJobQuery)).
					WithArgs(id).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)

				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, 999, notFoundErr.ID)
			},
		},
		{
			name: "database error",
			id:   1,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(deleteJobQuery)).
					WithArgs(id).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.ErrorIs(t, err, dbError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockDB.Close()

			repo := NewRepository(mockDB)
			tt.mockSetup(mockDB, tt.id)

			err = repo.Delete(context.Background(), tt.id)
			tt.checkResults(t, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetBySignature(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		signature    string
		mockSetup    func(mock pgxmock.PgxPoolIface, signature string)
		checkResults func(t *testing.T, result *Job, err error)
	}{
		{
			name:      "job found",
			signature: "job-signature-1",
			mockSetup: func(mock pgxmock.PgxPoolIface, signature string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobBySignatureQuery)).
					WithArgs(signature).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", "description", "experience_level", "employment_type",
						"location", "work_mode", "application_url", "is_active", "signature", "created_at", "updated_at",
					}).AddRow(
						1, 1, "Software Engineer", "Job description", "Mid-Level", "Full-Time",
						"San Francisco", "Remote", "https://example.com/apply", true, "job-signature-1", now, now,
					))
			},
			checkResults: func(t *testing.T, result *Job, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, 1, result.CompanyID)
				assert.Equal(t, "Software Engineer", result.Title)
				assert.Equal(t, "Job description", result.Description)
				assert.Equal(t, "Mid-Level", result.ExperienceLevel)
				assert.Equal(t, "Full-Time", result.EmploymentType)
				assert.Equal(t, "San Francisco", result.Location)
				assert.Equal(t, "Remote", result.WorkMode)
				assert.Equal(t, "https://example.com/apply", result.ApplicationURL)
				assert.True(t, result.IsActive)
				assert.Equal(t, "job-signature-1", result.Signature)
				assert.Equal(t, now, result.CreatedAt)
				assert.Equal(t, now, result.UpdatedAt)
			},
		},
		{
			name:      "job not found",
			signature: "nonexistent-signature",
			mockSetup: func(mock pgxmock.PgxPoolIface, signature string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobBySignatureQuery)).
					WithArgs(signature).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *Job, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)

				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, "nonexistent-signature", notFoundErr.Signature)
			},
		},
		{
			name:      "database error",
			signature: "error-signature",
			mockSetup: func(mock pgxmock.PgxPoolIface, signature string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobBySignatureQuery)).
					WithArgs(signature).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *Job, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
				require.ErrorIs(t, err, dbError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockDB.Close()

			repo := NewRepository(mockDB)
			tt.mockSetup(mockDB, tt.signature)

			result, err := repo.GetBySignature(context.Background(), tt.signature)
			tt.checkResults(t, result, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_SearchJobsWithCount(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")
	dateFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name         string
		params       SearchParams
		mockSetup    func(mock pgxmock.PgxPoolIface, params SearchParams)
		checkResults func(t *testing.T, jobs []*JobWithCompany, total int, err error)
	}{
		{
			name: "successful search with basic query",
			params: SearchParams{
				Query:  "software engineer",
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ SearchParams) {
				t.Helper()
				expectedQuery := searchJobsWithCountBaseQuery + " ORDER BY j.created_at DESC LIMIT $2 OFFSET $3"
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs("software engineer", 10, 0).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", "description", "experience_level", "employment_type",
						"location", "work_mode", "application_url", "is_active", "signature", "created_at", "updated_at",
						"company_name", "company_logo_url", "total_count",
					}).AddRow(
						1, 1, "Software Engineer", "Job description", "Mid-Level", "Full-Time",
						"San Francisco", "Remote", "https://example.com/apply", true, "job-signature-1", now, now,
						"Tech Corp", "https://example.com/logo1.png", 25,
					).AddRow(
						2, 2, "Senior Software Engineer", "Senior position", "Senior", "Full-Time",
						"New York", "Hybrid", "https://example.com/apply2", true, "job-signature-2", now, now,
						"Innovation Inc", "https://example.com/logo2.png", 25,
					))
			},
			checkResults: func(t *testing.T, jobs []*JobWithCompany, total int, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, jobs, 2)
				assert.Equal(t, 25, total)

				assert.Equal(t, "Software Engineer", jobs[0].Title)
				assert.Equal(t, "Tech Corp", jobs[0].CompanyName)
				assert.Equal(t, "https://example.com/logo1.png", jobs[0].CompanyLogoURL)

				assert.Equal(t, "Senior Software Engineer", jobs[1].Title)
				assert.Equal(t, "Innovation Inc", jobs[1].CompanyName)
				assert.Equal(t, "https://example.com/logo2.png", jobs[1].CompanyLogoURL)
			},
		},
		{
			name: "search with all filters applied",
			params: SearchParams{
				Query:           "developer",
				Limit:           5,
				Offset:          10,
				ExperienceLevel: stringPtr("Senior"),
				EmploymentType:  stringPtr("Full-Time"),
				Location:        stringPtr("San Francisco"),
				WorkMode:        stringPtr("Remote"),
				Company:         stringPtr("StartupXYZ"),
				DateFrom:        &dateFrom,
				DateTo:          &dateTo,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ SearchParams) {
				t.Helper()
				expectedQuery := searchJobsWithCountBaseQuery +
					" AND j.experience_level = $2 AND j.employment_type = $3 AND j.location = $4 AND j.work_mode = $5" +
					" AND LOWER(c.name) LIKE LOWER($6) AND j.created_at >= $7 AND j.created_at <= $8" +
					" ORDER BY j.created_at DESC LIMIT $9 OFFSET $10"
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs("developer", "Senior", "Full-Time", "San Francisco", "Remote", "%StartupXYZ%", dateFrom, dateTo, 5, 10).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", "description", "experience_level", "employment_type",
						"location", "work_mode", "application_url", "is_active", "signature", "created_at", "updated_at",
						"company_name", "company_logo_url", "total_count",
					}).AddRow(
						3, 3, "Senior Developer", "Senior developer position", "Senior", "Full-Time",
						"San Francisco", "Remote", "https://example.com/apply3", true, "job-signature-3", now, now,
						"StartupXYZ", "https://example.com/logo3.png", 42,
					))
			},
			checkResults: func(t *testing.T, jobs []*JobWithCompany, total int, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, jobs, 1)
				assert.Equal(t, 42, total)

				assert.Equal(t, "Senior Developer", jobs[0].Title)
				assert.Equal(t, "Senior", jobs[0].ExperienceLevel)
				assert.Equal(t, "Full-Time", jobs[0].EmploymentType)
				assert.Equal(t, "San Francisco", jobs[0].Location)
				assert.Equal(t, "Remote", jobs[0].WorkMode)
				assert.Equal(t, "StartupXYZ", jobs[0].CompanyName)
				assert.Equal(t, "https://example.com/logo3.png", jobs[0].CompanyLogoURL)
			},
		},
		{
			name: "search with no results",
			params: SearchParams{
				Query:  "nonexistent job title",
				Limit:  20,
				Offset: 0,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ SearchParams) {
				t.Helper()
				expectedQuery := searchJobsWithCountBaseQuery + " ORDER BY j.created_at DESC LIMIT $2 OFFSET $3"
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs("nonexistent job title", 20, 0).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", "description", "experience_level", "employment_type",
						"location", "work_mode", "application_url", "is_active", "signature", "created_at", "updated_at",
						"company_name", "company_logo_url", "total_count",
					}))
			},
			checkResults: func(t *testing.T, jobs []*JobWithCompany, total int, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, jobs)
				assert.Equal(t, 0, total)
			},
		},
		{
			name: "database error during search",
			params: SearchParams{
				Query:  "test query",
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ SearchParams) {
				t.Helper()
				expectedQuery := searchJobsWithCountBaseQuery + " ORDER BY j.created_at DESC LIMIT $2 OFFSET $3"
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs("test query", 10, 0).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, jobs []*JobWithCompany, total int, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, jobs)
				assert.Equal(t, 0, total)
				require.ErrorIs(t, err, dbError)
			},
		},
		{
			name: "empty query handled by database",
			params: SearchParams{
				Query:  "",
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ SearchParams) {
				t.Helper()
				expectedQuery := searchJobsWithCountBaseQuery + " ORDER BY j.created_at DESC LIMIT $2 OFFSET $3"
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs("", 10, 0).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", "description", "experience_level", "employment_type",
						"location", "work_mode", "application_url", "is_active", "signature", "created_at", "updated_at",
						"company_name", "company_logo_url", "total_count",
					}))
			},
			checkResults: func(t *testing.T, jobs []*JobWithCompany, total int, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, jobs)
				assert.Equal(t, 0, total)
			},
		},
		{
			name: "whitespace query trimmed and handled",
			params: SearchParams{
				Query:  "   ",
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ SearchParams) {
				t.Helper()
				expectedQuery := searchJobsWithCountBaseQuery + " ORDER BY j.created_at DESC LIMIT $2 OFFSET $3"
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs("", 10, 0). // Query should be trimmed to empty string
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", "description", "experience_level", "employment_type",
						"location", "work_mode", "application_url", "is_active", "signature", "created_at", "updated_at",
						"company_name", "company_logo_url", "total_count",
					}))
			},
			checkResults: func(t *testing.T, jobs []*JobWithCompany, total int, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, jobs)
				assert.Equal(t, 0, total)
			},
		},
		{
			name: "scan error in job rows",
			params: SearchParams{
				Query:  "test query",
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ SearchParams) {
				t.Helper()
				expectedQuery := searchJobsWithCountBaseQuery + " ORDER BY j.created_at DESC LIMIT $2 OFFSET $3"
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs("test query", 10, 0).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", // Missing columns to cause scan error
					}).AddRow(
						1, 1, "Software Engineer",
					))
			},
			checkResults: func(t *testing.T, jobs []*JobWithCompany, total int, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, jobs)
				assert.Equal(t, 0, total)
				assert.Contains(t, err.Error(), "scan")
			},
		},
		{
			name: "single result with correct total count",
			params: SearchParams{
				Query:  "golang",
				Limit:  1,
				Offset: 5,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ SearchParams) {
				t.Helper()
				expectedQuery := searchJobsWithCountBaseQuery + " ORDER BY j.created_at DESC LIMIT $2 OFFSET $3"
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs("golang", 1, 5).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "company_id", "title", "description", "experience_level", "employment_type",
						"location", "work_mode", "application_url", "is_active", "signature", "created_at", "updated_at",
						"company_name", "company_logo_url", "total_count",
					}).AddRow(
						6, 6, "Golang Developer", "Golang position", "Mid-level", "Full-Time",
						"Remote", "Remote", "https://example.com/apply6", true, "job-signature-6", now, now,
						"Go Corp", "https://example.com/logo6.png", 100,
					))
			},
			checkResults: func(t *testing.T, jobs []*JobWithCompany, total int, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, jobs, 1)
				assert.Equal(t, 100, total) // Total should be 100 even though we only returned 1 job
				assert.Equal(t, "Golang Developer", jobs[0].Title)
				assert.Equal(t, "Go Corp", jobs[0].CompanyName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockDB, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockDB.Close()

			repo := NewRepository(mockDB)
			tt.mockSetup(mockDB, tt.params)

			jobs, total, err := repo.SearchJobsWithCount(context.Background(), &tt.params)
			tt.checkResults(t, jobs, total, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
