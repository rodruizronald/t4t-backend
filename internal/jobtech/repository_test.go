package jobtech

import (
	"context"
	"errors"
	"fmt"
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
		jobTech      *JobTechnology
		mockSetup    func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology)
		checkResults func(t *testing.T, result *JobTechnology, err error)
	}{
		{
			name: "successful creation",
			jobTech: &JobTechnology{
				JobID:        1,
				TechnologyID: 2,
				IsRequired:   true,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createJobTechnologyQuery)).
					WithArgs(
						jobTech.JobID,
						jobTech.TechnologyID,
						jobTech.IsRequired,
					).
					WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))
			},
			checkResults: func(t *testing.T, result *JobTechnology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name: "duplicate job-technology association",
			jobTech: &JobTechnology{
				JobID:        1,
				TechnologyID: 2,
				IsRequired:   true,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology) {
				t.Helper()
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "job_technologies_job_id_technology_id_key",
				}
				mock.ExpectQuery(regexp.QuoteMeta(createJobTechnologyQuery)).
					WithArgs(
						jobTech.JobID,
						jobTech.TechnologyID,
						jobTech.IsRequired,
					).
					WillReturnError(pgErr)
			},
			checkResults: func(t *testing.T, _ *JobTechnology, err error) {
				t.Helper()
				require.Error(t, err)
				var duplicateErr *DuplicateError
				require.ErrorAs(t, err, &duplicateErr)
				assert.Equal(t, 1, duplicateErr.JobID)
				assert.Equal(t, 2, duplicateErr.TechnologyID)
			},
		},
		{
			name: "database error",
			jobTech: &JobTechnology{
				JobID:        1,
				TechnologyID: 2,
				IsRequired:   true,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createJobTechnologyQuery)).
					WithArgs(
						jobTech.JobID,
						jobTech.TechnologyID,
						jobTech.IsRequired,
					).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, _ *JobTechnology, err error) {
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
			tt.mockSetup(mockDB, tt.jobTech)

			err = repo.Create(context.Background(), tt.jobTech)
			tt.checkResults(t, tt.jobTech, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetByJobAndTechnology(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		jobID        int
		techID       int
		mockSetup    func(mock pgxmock.PgxPoolIface, jobID, techID int)
		checkResults func(t *testing.T, result *JobTechnology, err error)
	}{
		{
			name:   "association found",
			jobID:  1,
			techID: 2,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID, techID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobTechnologyByJobAndTechQuery)).
					WithArgs(jobID, techID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", "is_required", "created_at",
					}).AddRow(
						1, jobID, techID, true, now,
					))
			},
			checkResults: func(t *testing.T, result *JobTechnology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, 1, result.JobID)
				assert.Equal(t, 2, result.TechnologyID)
				assert.True(t, result.IsRequired)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name:   "association not found",
			jobID:  999,
			techID: 888,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID, techID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobTechnologyByJobAndTechQuery)).
					WithArgs(jobID, techID).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *JobTechnology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, 999, notFoundErr.JobID)
				assert.Equal(t, 888, notFoundErr.TechnologyID)
			},
		},
		{
			name:   "database error",
			jobID:  1,
			techID: 2,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID, techID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getJobTechnologyByJobAndTechQuery)).
					WithArgs(jobID, techID).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *JobTechnology, err error) {
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
			tt.mockSetup(mockDB, tt.jobID, tt.techID)

			result, err := repo.GetByJobAndTechnology(context.Background(), tt.jobID, tt.techID)
			tt.checkResults(t, result, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_Update(t *testing.T) {
	t.Parallel()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		jobTech      *JobTechnology
		mockSetup    func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology)
		checkResults func(t *testing.T, err error)
	}{
		{
			name: "successful update",
			jobTech: &JobTechnology{
				ID:           1,
				JobID:        1,
				TechnologyID: 2,
				IsRequired:   false,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateJobTechnologyQuery)).
					WithArgs(
						jobTech.IsRequired,
						jobTech.ID,
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "job technology not found",
			jobTech: &JobTechnology{
				ID:           999,
				JobID:        1,
				TechnologyID: 2,
				IsRequired:   false,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateJobTechnologyQuery)).
					WithArgs(
						jobTech.IsRequired,
						jobTech.ID,
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
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
			name: "duplicate constraint error",
			jobTech: &JobTechnology{
				ID:           1,
				JobID:        1,
				TechnologyID: 2,
				IsRequired:   false,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology) {
				t.Helper()
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "job_technologies_job_id_technology_id_key",
				}
				mock.ExpectExec(regexp.QuoteMeta(updateJobTechnologyQuery)).
					WithArgs(
						jobTech.IsRequired,
						jobTech.ID,
					).
					WillReturnError(pgErr)
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				var duplicateErr *DuplicateError
				require.ErrorAs(t, err, &duplicateErr)
				assert.Equal(t, 1, duplicateErr.JobID)
				assert.Equal(t, 2, duplicateErr.TechnologyID)
			},
		},
		{
			name: "database error",
			jobTech: &JobTechnology{
				ID:           1,
				JobID:        1,
				TechnologyID: 2,
				IsRequired:   false,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, jobTech *JobTechnology) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateJobTechnologyQuery)).
					WithArgs(
						jobTech.IsRequired,
						jobTech.ID,
					).
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
			tt.mockSetup(mockDB, tt.jobTech)

			err = repo.Update(context.Background(), tt.jobTech)
			tt.checkResults(t, err)

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
			name: "successful delete",
			id:   1,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(deleteJobTechnologyQuery)).
					WithArgs(id).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "job technology not found",
			id:   999,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(deleteJobTechnologyQuery)).
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
				mock.ExpectExec(regexp.QuoteMeta(deleteJobTechnologyQuery)).
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

func TestRepository_ListByJob(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		jobID        int
		mockSetup    func(mock pgxmock.PgxPoolIface, jobID int)
		checkResults func(t *testing.T, results []*JobTechnology, err error)
	}{
		{
			name:  "successful listing with results",
			jobID: 1,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listJobTechnologiesByJobQuery)).
					WithArgs(jobID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", "is_required", "created_at",
					}).AddRow(
						1, jobID, 2, true, now,
					).AddRow(
						2, jobID, 3, true, now,
					))
			},
			checkResults: func(t *testing.T, results []*JobTechnology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, results, 2)
				assert.Equal(t, 1, results[0].ID)
				assert.Equal(t, 1, results[0].JobID)
				assert.Equal(t, 2, results[0].TechnologyID)
				assert.True(t, results[0].IsRequired)
				assert.Equal(t, 2, results[1].ID)
				assert.Equal(t, 1, results[1].JobID)
				assert.Equal(t, 3, results[1].TechnologyID)
				assert.True(t, results[1].IsRequired)
			},
		},
		{
			name:  "successful listing with no results",
			jobID: 999,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listJobTechnologiesByJobQuery)).
					WithArgs(jobID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", "is_required", "created_at",
					}))
			},
			checkResults: func(t *testing.T, results []*JobTechnology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, results)
			},
		},
		{
			name:  "database error",
			jobID: 1,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listJobTechnologiesByJobQuery)).
					WithArgs(jobID).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, results []*JobTechnology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, results)
				require.ErrorIs(t, err, dbError)
			},
		},
		{
			name:  "scan error",
			jobID: 1,
			mockSetup: func(mock pgxmock.PgxPoolIface, jobID int) {
				t.Helper()
				// Return mismatched column count to cause scan error
				mock.ExpectQuery(regexp.QuoteMeta(listJobTechnologiesByJobQuery)).
					WithArgs(jobID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", // Missing columns to cause scan error
					}).AddRow(
						1, jobID,
					))
			},
			checkResults: func(t *testing.T, results []*JobTechnology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, results)
				assert.Contains(t, err.Error(), "scan")
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

			results, err := repo.ListByJob(context.Background(), tt.jobID)
			tt.checkResults(t, results, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_ListByTechnology(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		techID       int
		mockSetup    func(mock pgxmock.PgxPoolIface, techID int)
		checkResults func(t *testing.T, results []*JobTechnology, err error)
	}{
		{
			name:   "successful listing with results",
			techID: 2,
			mockSetup: func(mock pgxmock.PgxPoolIface, techID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listJobTechnologiesByTechnologyQuery)).
					WithArgs(techID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", "is_required", "created_at",
					}).AddRow(
						1, 1, techID, true, now,
					).AddRow(
						3, 2, techID, true, now,
					))
			},
			checkResults: func(t *testing.T, results []*JobTechnology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, results, 2)
				assert.Equal(t, 1, results[0].ID)
				assert.Equal(t, 1, results[0].JobID)
				assert.Equal(t, 2, results[0].TechnologyID)
				assert.True(t, results[0].IsRequired)
				assert.Equal(t, 3, results[1].ID)
				assert.Equal(t, 2, results[1].JobID)
				assert.Equal(t, 2, results[1].TechnologyID)
				assert.True(t, results[1].IsRequired)
			},
		},
		{
			name:   "successful listing with no results",
			techID: 999,
			mockSetup: func(mock pgxmock.PgxPoolIface, techID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listJobTechnologiesByTechnologyQuery)).
					WithArgs(techID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", "is_required", "created_at",
					}))
			},
			checkResults: func(t *testing.T, results []*JobTechnology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, results)
			},
		},
		{
			name:   "database error",
			techID: 2,
			mockSetup: func(mock pgxmock.PgxPoolIface, techID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listJobTechnologiesByTechnologyQuery)).
					WithArgs(techID).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, results []*JobTechnology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, results)
				require.ErrorIs(t, err, dbError)
			},
		},
		{
			name:   "scan error",
			techID: 2,
			mockSetup: func(mock pgxmock.PgxPoolIface, techID int) {
				t.Helper()
				// Return mismatched column count to cause scan error
				mock.ExpectQuery(regexp.QuoteMeta(listJobTechnologiesByTechnologyQuery)).
					WithArgs(techID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", // Missing columns to cause scan error
					}).AddRow(
						1, 1,
					))
			},
			checkResults: func(t *testing.T, results []*JobTechnology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, results)
				assert.Contains(t, err.Error(), "scan")
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
			tt.mockSetup(mockDB, tt.techID)

			results, err := repo.ListByTechnology(context.Background(), tt.techID)
			tt.checkResults(t, results, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetJobTechnologiesBatch(t *testing.T) {
	t.Parallel()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		jobIDs       []int
		mockSetup    func(mock pgxmock.PgxPoolIface, jobIDs []int)
		checkResults func(t *testing.T, results map[int][]*JobTechnologyWithDetails, err error)
	}{
		{
			name:   "successful batch retrieval with multiple jobs",
			jobIDs: []int{1, 2},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ []int) {
				t.Helper()
				expectedQuery := fmt.Sprintf(getJobTechnologiesBatchQuery, "$1,$2")
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs(1, 2).
					WillReturnRows(pgxmock.NewRows([]string{
						"job_id", "technology_id", "is_required", "tech_name", "tech_category",
					}).AddRow(
						1, 10, true, "Go", "Programming Language",
					).AddRow(
						1, 11, false, "PostgreSQL", "Database",
					).AddRow(
						2, 10, true, "Go", "Programming Language",
					).AddRow(
						2, 12, true, "React", "Framework",
					))
			},
			checkResults: func(t *testing.T, results map[int][]*JobTechnologyWithDetails, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, results, 2)

				// Check job 1 technologies
				job1Techs := results[1]
				assert.Len(t, job1Techs, 2)
				assert.Equal(t, 1, job1Techs[0].JobID)
				assert.Equal(t, 10, job1Techs[0].TechnologyID)
				assert.Equal(t, "Go", job1Techs[0].TechName)
				assert.Equal(t, "Programming Language", job1Techs[0].TechCategory)
				assert.True(t, job1Techs[0].IsRequired)

				assert.Equal(t, 1, job1Techs[1].JobID)
				assert.Equal(t, 11, job1Techs[1].TechnologyID)
				assert.Equal(t, "PostgreSQL", job1Techs[1].TechName)
				assert.Equal(t, "Database", job1Techs[1].TechCategory)
				assert.False(t, job1Techs[1].IsRequired)

				// Check job 2 technologies
				job2Techs := results[2]
				assert.Len(t, job2Techs, 2)
				assert.Equal(t, 2, job2Techs[0].JobID)
				assert.Equal(t, 10, job2Techs[0].TechnologyID)
				assert.Equal(t, "Go", job2Techs[0].TechName)
				assert.Equal(t, "Programming Language", job2Techs[0].TechCategory)
				assert.True(t, job2Techs[0].IsRequired)

				assert.Equal(t, 2, job2Techs[1].JobID)
				assert.Equal(t, 12, job2Techs[1].TechnologyID)
				assert.Equal(t, "React", job2Techs[1].TechName)
				assert.Equal(t, "Framework", job2Techs[1].TechCategory)
				assert.True(t, job2Techs[1].IsRequired)
			},
		},
		{
			name:   "successful batch retrieval with single job",
			jobIDs: []int{1},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ []int) {
				t.Helper()
				expectedQuery := fmt.Sprintf(getJobTechnologiesBatchQuery, "$1")
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs(1).
					WillReturnRows(pgxmock.NewRows([]string{
						"job_id", "technology_id", "is_required", "tech_name", "tech_category",
					}).AddRow(
						1, 10, true, "Go", "Programming Language",
					))
			},
			checkResults: func(t *testing.T, results map[int][]*JobTechnologyWithDetails, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, results, 1)

				job1Techs := results[1]
				assert.Len(t, job1Techs, 1)
				assert.Equal(t, 1, job1Techs[0].JobID)
				assert.Equal(t, 10, job1Techs[0].TechnologyID)
				assert.Equal(t, "Go", job1Techs[0].TechName)
				assert.Equal(t, "Programming Language", job1Techs[0].TechCategory)
				assert.True(t, job1Techs[0].IsRequired)
			},
		},
		{
			name:   "empty job IDs slice",
			jobIDs: []int{},
			mockSetup: func(_ pgxmock.PgxPoolIface, _ []int) {
				t.Helper()
				// No database call expected for empty slice
			},
			checkResults: func(t *testing.T, results map[int][]*JobTechnologyWithDetails, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, results)
			},
		},
		{
			name:   "jobs with no technologies",
			jobIDs: []int{999, 888},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ []int) {
				t.Helper()
				expectedQuery := fmt.Sprintf(getJobTechnologiesBatchQuery, "$1,$2")
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs(999, 888).
					WillReturnRows(pgxmock.NewRows([]string{
						"job_id", "technology_id", "is_required", "tech_name", "tech_category",
					}))
			},
			checkResults: func(t *testing.T, results map[int][]*JobTechnologyWithDetails, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, results)
			},
		},
		{
			name:   "database error",
			jobIDs: []int{1, 2},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ []int) {
				t.Helper()
				expectedQuery := fmt.Sprintf(getJobTechnologiesBatchQuery, "$1,$2")
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs(1, 2).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, results map[int][]*JobTechnologyWithDetails, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, results)
				require.ErrorIs(t, err, dbError)
			},
		},
		{
			name:   "scan error",
			jobIDs: []int{1},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ []int) {
				t.Helper()
				expectedQuery := fmt.Sprintf(getJobTechnologiesBatchQuery, "$1")
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs(1).
					WillReturnRows(pgxmock.NewRows([]string{
						"job_id", "technology_id", // Missing columns to cause scan error
					}).AddRow(
						1, 10,
					))
			},
			checkResults: func(t *testing.T, results map[int][]*JobTechnologyWithDetails, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, results)
				assert.Contains(t, err.Error(), "scan")
			},
		},
		{
			name:   "partial results - some jobs have technologies, others don't",
			jobIDs: []int{1, 2, 3},
			mockSetup: func(mock pgxmock.PgxPoolIface, _ []int) {
				t.Helper()
				expectedQuery := fmt.Sprintf(getJobTechnologiesBatchQuery, "$1,$2,$3")
				mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
					WithArgs(1, 2, 3).
					WillReturnRows(pgxmock.NewRows([]string{
						"job_id", "technology_id", "is_required", "tech_name", "tech_category",
					}).AddRow(
						1, 10, true, "Go", "Programming Language",
					).AddRow(
						3, 12, false, "React", "Framework",
					))
			},
			checkResults: func(t *testing.T, results map[int][]*JobTechnologyWithDetails, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, results, 2) // Only jobs 1 and 3 have technologies

				// Job 1 should have technologies
				job1Techs := results[1]
				assert.Len(t, job1Techs, 1)
				assert.Equal(t, 1, job1Techs[0].JobID)
				assert.Equal(t, 10, job1Techs[0].TechnologyID)

				// Job 2 should not be in results (no technologies)
				_, exists := results[2]
				assert.False(t, exists)

				// Job 3 should have technologies
				job3Techs := results[3]
				assert.Len(t, job3Techs, 1)
				assert.Equal(t, 3, job3Techs[0].JobID)
				assert.Equal(t, 12, job3Techs[0].TechnologyID)
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
			tt.mockSetup(mockDB, tt.jobIDs)

			results, err := repo.GetJobTechnologiesBatch(context.Background(), tt.jobIDs)
			tt.checkResults(t, results, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}
