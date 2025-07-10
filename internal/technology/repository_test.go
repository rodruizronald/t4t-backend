package technology

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
	parentID := 5

	tests := []struct {
		name         string
		technology   *Technology
		mockSetup    func(mock pgxmock.PgxPoolIface, technology *Technology)
		checkResults func(t *testing.T, result *Technology, err error)
	}{
		{
			name: "successful creation",
			technology: &Technology{
				Name:     "Go",
				Category: "Programming Language",
				ParentID: nil,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).
						AddRow(1, now))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name: "successful creation with parent ID",
			technology: &Technology{
				Name:     "Gin",
				Category: "Framework",
				ParentID: &parentID,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).
						AddRow(2, now))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, 2, result.ID)
				assert.Equal(t, now, result.CreatedAt)
				assert.Equal(t, &parentID, result.ParentID)
			},
		},
		{
			name: "duplicate technology name",
			technology: &Technology{
				Name:     "Duplicate Tech",
				Category: "Programming Language",
				ParentID: nil,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID).
					WillReturnError(&pgconn.PgError{Code: "23505"})
			},
			checkResults: func(t *testing.T, _ *Technology, err error) {
				t.Helper()
				var duplicateErr *DuplicateError
				require.Error(t, err)
				require.ErrorAs(t, err, &duplicateErr)
				assert.Equal(t, "Duplicate Tech", duplicateErr.Name)
			},
		},
		{
			name: "database error",
			technology: &Technology{
				Name:     "Error Tech",
				Category: "Programming Language",
				ParentID: nil,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, _ *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.ErrorIs(t, err, dbError)
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
			tt.mockSetup(mockDB, tt.technology)

			err = repo.Create(context.Background(), tt.technology)
			tt.checkResults(t, tt.technology, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetByID(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")
	parentID := 5

	tests := []struct {
		name         string
		id           int
		mockSetup    func(mock pgxmock.PgxPoolIface, id int)
		checkResults func(t *testing.T, result *Technology, err error)
	}{
		{
			name: "technology found",
			id:   1,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Go", "Programming Language", nil, now,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, "Go", result.Name)
				assert.Equal(t, "Programming Language", result.Category)
				assert.Nil(t, result.ParentID)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name: "technology found with parent ID",
			id:   2,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Gin", "Framework", &parentID, now,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 2, result.ID)
				assert.Equal(t, "Gin", result.Name)
				assert.Equal(t, "Framework", result.Category)
				assert.Equal(t, parentID, *result.ParentID)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name: "technology not found",
			id:   999,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)

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
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
				assert.ErrorIs(t, err, dbError)
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

			result, err := repo.GetByID(context.Background(), tt.id)
			tt.checkResults(t, result, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetByName(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")
	parentID := 5

	tests := []struct {
		name         string
		techName     string
		mockSetup    func(mock pgxmock.PgxPoolIface, techName string)
		checkResults func(t *testing.T, result *Technology, err error)
	}{
		{
			name:     "technology found",
			techName: "Go",
			mockSetup: func(mock pgxmock.PgxPoolIface, techName string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByNameQuery)).
					WithArgs(techName).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						1, techName, "Programming Language", nil, now,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, "Go", result.Name)
				assert.Equal(t, "Programming Language", result.Category)
				assert.Nil(t, result.ParentID)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name:     "technology found with parent ID",
			techName: "Gin",
			mockSetup: func(mock pgxmock.PgxPoolIface, techName string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByNameQuery)).
					WithArgs(techName).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						2, techName, "Framework", &parentID, now,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 2, result.ID)
				assert.Equal(t, "Gin", result.Name)
				assert.Equal(t, "Framework", result.Category)
				assert.NotNil(t, result.ParentID)
				assert.Equal(t, parentID, *result.ParentID)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name:     "technology not found",
			techName: "NonexistentTech",
			mockSetup: func(mock pgxmock.PgxPoolIface, techName string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByNameQuery)).
					WithArgs(techName).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)

				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, "NonexistentTech", notFoundErr.Name)
			},
		},
		{
			name:     "database error",
			techName: "ErrorTech",
			mockSetup: func(mock pgxmock.PgxPoolIface, techName string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByNameQuery)).
					WithArgs(techName).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
				assert.ErrorIs(t, err, dbError)
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
			tt.mockSetup(mockDB, tt.techName)

			result, err := repo.GetByName(context.Background(), tt.techName)
			tt.checkResults(t, result, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_Update(t *testing.T) {
	t.Parallel()
	dbError := errors.New("database error")
	parentID := 5

	tests := []struct {
		name         string
		technology   *Technology
		mockSetup    func(mock pgxmock.PgxPoolIface, technology *Technology)
		checkResults func(t *testing.T, result *Technology, err error)
	}{
		{
			name: "successful update without parent ID",
			technology: &Technology{
				ID:       1,
				Name:     "Updated Tech",
				Category: "Updated Category",
				ParentID: nil,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID, technology.ID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			checkResults: func(t *testing.T, _ *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "successful update with parent ID",
			technology: &Technology{
				ID:       2,
				Name:     "Updated Framework",
				Category: "Framework",
				ParentID: &parentID,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID, technology.ID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, &parentID, result.ParentID)
			},
		},
		{
			name: "technology not found",
			technology: &Technology{
				ID:       999,
				Name:     "Nonexistent Tech",
				Category: "Nonexistent Category",
				ParentID: nil,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID, technology.ID).
					WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			},
			checkResults: func(t *testing.T, _ *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, 999, notFoundErr.ID)
			},
		},
		{
			name: "duplicate technology name",
			technology: &Technology{
				ID:       3,
				Name:     "Duplicate Tech",
				Category: "Programming Language",
				ParentID: nil,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "technologies_name_key",
				}
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID, technology.ID).
					WillReturnError(pgErr)
			},
			checkResults: func(t *testing.T, _ *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				var duplicateErr *DuplicateError
				require.ErrorAs(t, err, &duplicateErr)
				assert.Equal(t, "Duplicate Tech", duplicateErr.Name)
			},
		},
		{
			name: "database error",
			technology: &Technology{
				ID:       1,
				Name:     "Error Tech",
				Category: "Error Category",
				ParentID: nil,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, technology *Technology) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyQuery)).
					WithArgs(technology.Name, technology.Category, technology.ParentID, technology.ID).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, _ *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.ErrorIs(t, err, dbError)
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
			tt.mockSetup(mockDB, tt.technology)

			err = repo.Update(context.Background(), tt.technology)
			tt.checkResults(t, tt.technology, err)

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
				mock.ExpectExec(regexp.QuoteMeta(deleteTechnologyQuery)).
					WithArgs(id).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "technology not found",
			id:   999,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(deleteTechnologyQuery)).
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
				mock.ExpectExec(regexp.QuoteMeta(deleteTechnologyQuery)).
					WithArgs(id).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.ErrorIs(t, err, dbError)
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

func TestRepository_GetWithAliases(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")
	parentID := 5

	tests := []struct {
		name         string
		id           int
		mockSetup    func(mock pgxmock.PgxPoolIface, id int)
		checkResults func(t *testing.T, result *Technology, err error)
	}{
		{
			name: "successful retrieval with aliases",
			id:   1,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "JavaScript", "Programming Language", nil, now,
					))

				// Second query to get the aliases
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasesQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", "alias", "created_at",
					}).AddRow(
						1, id, "JS", now,
					).AddRow(
						2, id, "ECMAScript", now,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, "JavaScript", result.Name)
				assert.Equal(t, "Programming Language", result.Category)
				assert.Nil(t, result.ParentID)

				// Check aliases
				assert.Len(t, result.Aliases, 2)
				assert.Equal(t, "JS", result.Aliases[0].Alias)
				assert.Equal(t, "ECMAScript", result.Aliases[1].Alias)
			},
		},
		{
			name: "successful retrieval with parent ID and aliases",
			id:   2,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "React", "Framework", &parentID, now,
					))

				// Second query to get the aliases
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasesQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", "alias", "created_at",
					}).AddRow(
						3, id, "ReactJS", now,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 2, result.ID)
				assert.Equal(t, "React", result.Name)
				assert.Equal(t, "Framework", result.Category)
				assert.NotNil(t, result.ParentID)
				assert.Equal(t, parentID, *result.ParentID)

				// Check aliases
				assert.Len(t, result.Aliases, 1)
				assert.Equal(t, "ReactJS", result.Aliases[0].Alias)
			},
		},
		{
			name: "technology not found",
			id:   999,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)

				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, 999, notFoundErr.ID)
			},
		},
		{
			name: "technology found but error fetching aliases",
			id:   3,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Python", "Programming Language", nil, now,
					))

				// Second query to get aliases returns error
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasesQuery)).
					WithArgs(id).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
				assert.ErrorIs(t, err, dbError)
			},
		},
		{
			name: "technology found with no aliases",
			id:   4,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Go", "Programming Language", nil, now,
					))

				// Second query to get aliases returns empty result
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasesQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", "alias", "created_at",
					}))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 4, result.ID)
				assert.Equal(t, "Go", result.Name)
				assert.Equal(t, "Programming Language", result.Category)
				assert.Nil(t, result.ParentID)
				assert.Empty(t, result.Aliases)
			},
		},
		{
			name: "scan error in aliases",
			id:   5,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Ruby", "Programming Language", nil, now,
					))

				// Second query returns mismatched columns to cause scan error
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasesQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", // Missing columns to cause scan error
					}).AddRow(
						1, id,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
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
			tt.mockSetup(mockDB, tt.id)

			result, err := repo.GetWithAliases(context.Background(), tt.id)
			tt.checkResults(t, result, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetWithJobs(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")
	parentID := 5

	tests := []struct {
		name         string
		id           int
		mockSetup    func(mock pgxmock.PgxPoolIface, id int)
		checkResults func(t *testing.T, result *Technology, err error)
	}{
		{
			name: "successful retrieval with jobs",
			id:   1,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Go", "Programming Language", nil, now,
					))

				// Second query to get the job associations
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyJobsQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", "is_required", "created_at",
					}).AddRow(
						1, 101, id, true, now,
					).AddRow(
						2, 102, id, true, now,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, "Go", result.Name)
				assert.Equal(t, "Programming Language", result.Category)
				assert.Nil(t, result.ParentID)

				// Check job associations
				assert.Len(t, result.Jobs, 2)
				assert.Equal(t, 101, result.Jobs[0].JobID)
				assert.Equal(t, 102, result.Jobs[1].JobID)
			},
		},
		{
			name: "successful retrieval with parent ID and jobs",
			id:   2,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "React", "Framework", &parentID, now,
					))

				// Second query to get the job associations
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyJobsQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", "is_required", "created_at",
					}).AddRow(
						3, 201, id, false, now,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 2, result.ID)
				assert.Equal(t, "React", result.Name)
				assert.Equal(t, "Framework", result.Category)
				assert.NotNil(t, result.ParentID)
				assert.Equal(t, parentID, *result.ParentID)

				// Check job associations
				assert.Len(t, result.Jobs, 1)
				assert.Equal(t, 201, result.Jobs[0].JobID)
				assert.False(t, result.Jobs[0].IsRequired)
			},
		},
		{
			name: "technology not found",
			id:   999,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)

				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, 999, notFoundErr.ID)
			},
		},
		{
			name: "technology found but error fetching jobs",
			id:   3,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Python", "Programming Language", nil, now,
					))

				// Second query to get jobs returns error
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyJobsQuery)).
					WithArgs(id).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
				assert.ErrorIs(t, err, dbError)
			},
		},
		{
			name: "technology found with no jobs",
			id:   4,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Ruby", "Programming Language", nil, now,
					))

				// Second query to get jobs returns empty result
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyJobsQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", "is_primary", "is_required", "created_at",
					}))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 4, result.ID)
				assert.Equal(t, "Ruby", result.Name)
				assert.Equal(t, "Programming Language", result.Category)
				assert.Nil(t, result.ParentID)
				assert.Empty(t, result.Jobs)
			},
		},
		{
			name: "scan error in jobs",
			id:   5,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				// First query to get the technology
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "name", "category", "parent_id", "created_at",
					}).AddRow(
						id, "Java", "Programming Language", nil, now,
					))

				// Second query returns mismatched columns to cause scan error
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyJobsQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "job_id", "technology_id", // Missing columns to cause scan error
					}).AddRow(
						1, 103, id,
					))
			},
			checkResults: func(t *testing.T, result *Technology, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)
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
			tt.mockSetup(mockDB, tt.id)

			result, err := repo.GetWithJobs(context.Background(), tt.id)
			tt.checkResults(t, result, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}
