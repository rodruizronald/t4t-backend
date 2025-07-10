package techalias

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
		alias        *TechnologyAlias
		mockSetup    func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias)
		checkResults func(t *testing.T, result *TechnologyAlias, err error)
	}{
		{
			name: "successful creation",
			alias: &TechnologyAlias{
				TechnologyID: 1,
				Alias:        "JS",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createTechnologyAliasQuery)).
					WithArgs(
						alias.TechnologyID,
						alias.Alias,
					).
					WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))
			},
			checkResults: func(t *testing.T, result *TechnologyAlias, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name: "duplicate alias",
			alias: &TechnologyAlias{
				TechnologyID: 1,
				Alias:        "JS",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias) {
				t.Helper()
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "technology_aliases_alias_key",
				}
				mock.ExpectQuery(regexp.QuoteMeta(createTechnologyAliasQuery)).
					WithArgs(
						alias.TechnologyID,
						alias.Alias,
					).
					WillReturnError(pgErr)
			},
			checkResults: func(t *testing.T, _ *TechnologyAlias, err error) {
				t.Helper()
				require.Error(t, err)

				var duplicateErr *DuplicateError
				require.ErrorAs(t, err, &duplicateErr)
				assert.Equal(t, "JS", duplicateErr.Alias)
			},
		},
		{
			name: "database error",
			alias: &TechnologyAlias{
				TechnologyID: 1,
				Alias:        "JS",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(createTechnologyAliasQuery)).
					WithArgs(
						alias.TechnologyID,
						alias.Alias,
					).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, _ *TechnologyAlias, err error) {
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
			tt.mockSetup(mockDB, tt.alias)

			err = repo.Create(context.Background(), tt.alias)
			tt.checkResults(t, tt.alias, err)

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
		id           int
		mockSetup    func(mock pgxmock.PgxPoolIface, id int)
		checkResults func(t *testing.T, result *TechnologyAlias, err error)
	}{
		{
			name: "alias found",
			id:   1,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasByIDQuery)).
					WithArgs(id).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", "alias", "created_at",
					}).AddRow(
						id, 1, "JS", now,
					))
			},
			checkResults: func(t *testing.T, result *TechnologyAlias, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, 1, result.TechnologyID)
				assert.Equal(t, "JS", result.Alias)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name: "alias not found",
			id:   999,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasByIDQuery)).
					WithArgs(id).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *TechnologyAlias, err error) {
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
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasByIDQuery)).
					WithArgs(id).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *TechnologyAlias, err error) {
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
			tt.mockSetup(mockDB, tt.id)

			result, err := repo.GetByID(context.Background(), tt.id)
			tt.checkResults(t, result, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetByAlias(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		alias        string
		mockSetup    func(mock pgxmock.PgxPoolIface, alias string)
		checkResults func(t *testing.T, result *TechnologyAlias, err error)
	}{
		{
			name:  "alias found",
			alias: "JS",
			mockSetup: func(mock pgxmock.PgxPoolIface, alias string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasByAliasQuery)).
					WithArgs(alias).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", "alias", "created_at",
					}).AddRow(
						1, 1, alias, now,
					))
			},
			checkResults: func(t *testing.T, result *TechnologyAlias, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, 1, result.TechnologyID)
				assert.Equal(t, "JS", result.Alias)
				assert.Equal(t, now, result.CreatedAt)
			},
		},
		{
			name:  "alias not found",
			alias: "NonExistentAlias",
			mockSetup: func(mock pgxmock.PgxPoolIface, alias string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasByAliasQuery)).
					WithArgs(alias).
					WillReturnError(pgx.ErrNoRows)
			},
			checkResults: func(t *testing.T, result *TechnologyAlias, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, result)

				var notFoundErr *NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
				assert.Equal(t, "NonExistentAlias", notFoundErr.Alias)
			},
		},
		{
			name:  "database error",
			alias: "JS",
			mockSetup: func(mock pgxmock.PgxPoolIface, alias string) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(getTechnologyAliasByAliasQuery)).
					WithArgs(alias).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, result *TechnologyAlias, err error) {
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
			tt.mockSetup(mockDB, tt.alias)

			result, err := repo.GetByAlias(context.Background(), tt.alias)
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
		alias        *TechnologyAlias
		mockSetup    func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias)
		checkResults func(t *testing.T, err error)
	}{
		{
			name: "successful update",
			alias: &TechnologyAlias{
				ID:           1,
				TechnologyID: 1,
				Alias:        "JavaScript",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyAliasQuery)).
					WithArgs(
						alias.Alias,
						alias.ID,
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "alias not found",
			alias: &TechnologyAlias{
				ID:           999,
				TechnologyID: 1,
				Alias:        "JavaScript",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyAliasQuery)).
					WithArgs(
						alias.Alias,
						alias.ID,
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
			name: "duplicate alias",
			alias: &TechnologyAlias{
				ID:           1,
				TechnologyID: 1,
				Alias:        "JS",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias) {
				t.Helper()
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "technology_aliases_alias_key",
				}
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyAliasQuery)).
					WithArgs(
						alias.Alias,
						alias.ID,
					).
					WillReturnError(pgErr)
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)

				var duplicateErr *DuplicateError
				require.ErrorAs(t, err, &duplicateErr)
				assert.Equal(t, "JS", duplicateErr.Alias)
			},
		},
		{
			name: "database error",
			alias: &TechnologyAlias{
				ID:           1,
				TechnologyID: 1,
				Alias:        "JS",
			},
			mockSetup: func(mock pgxmock.PgxPoolIface, alias *TechnologyAlias) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(updateTechnologyAliasQuery)).
					WithArgs(
						alias.Alias,
						alias.ID,
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
			tt.mockSetup(mockDB, tt.alias)

			err = repo.Update(context.Background(), tt.alias)
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
			name: "successful deletion",
			id:   1,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(deleteTechnologyAliasQuery)).
					WithArgs(id).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			checkResults: func(t *testing.T, err error) {
				t.Helper()
				require.NoError(t, err)
			},
		},
		{
			name: "alias not found",
			id:   999,
			mockSetup: func(mock pgxmock.PgxPoolIface, id int) {
				t.Helper()
				mock.ExpectExec(regexp.QuoteMeta(deleteTechnologyAliasQuery)).
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
				mock.ExpectExec(regexp.QuoteMeta(deleteTechnologyAliasQuery)).
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

func TestRepository_ListByTechnologyID(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dbError := errors.New("database error")

	tests := []struct {
		name         string
		technologyID int
		mockSetup    func(mock pgxmock.PgxPoolIface, technologyID int)
		checkResults func(t *testing.T, results []*TechnologyAlias, err error)
	}{
		{
			name:         "successful listing with results",
			technologyID: 1,
			mockSetup: func(mock pgxmock.PgxPoolIface, technologyID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listTechnologyAliasesByTechnologyIDQuery)).
					WithArgs(technologyID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", "alias", "created_at",
					}).AddRow(
						1, technologyID, "JS", now,
					).AddRow(
						2, technologyID, "JavaScript", now,
					))
			},
			checkResults: func(t *testing.T, results []*TechnologyAlias, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Len(t, results, 2)

				assert.Equal(t, 1, results[0].ID)
				assert.Equal(t, 1, results[0].TechnologyID)
				assert.Equal(t, "JS", results[0].Alias)
				assert.Equal(t, now, results[0].CreatedAt)

				assert.Equal(t, 2, results[1].ID)
				assert.Equal(t, 1, results[1].TechnologyID)
				assert.Equal(t, "JavaScript", results[1].Alias)
				assert.Equal(t, now, results[1].CreatedAt)
			},
		},
		{
			name:         "successful listing with no results",
			technologyID: 999,
			mockSetup: func(mock pgxmock.PgxPoolIface, technologyID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listTechnologyAliasesByTechnologyIDQuery)).
					WithArgs(technologyID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", "alias", "created_at",
					}))
			},
			checkResults: func(t *testing.T, results []*TechnologyAlias, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Empty(t, results)
			},
		},
		{
			name:         "database error",
			technologyID: 1,
			mockSetup: func(mock pgxmock.PgxPoolIface, technologyID int) {
				t.Helper()
				mock.ExpectQuery(regexp.QuoteMeta(listTechnologyAliasesByTechnologyIDQuery)).
					WithArgs(technologyID).
					WillReturnError(dbError)
			},
			checkResults: func(t *testing.T, results []*TechnologyAlias, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Nil(t, results)
				require.ErrorIs(t, err, dbError)
			},
		},
		{
			name:         "scan error",
			technologyID: 1,
			mockSetup: func(mock pgxmock.PgxPoolIface, technologyID int) {
				t.Helper()
				// Return mismatched column count to cause scan error
				mock.ExpectQuery(regexp.QuoteMeta(listTechnologyAliasesByTechnologyIDQuery)).
					WithArgs(technologyID).
					WillReturnRows(pgxmock.NewRows([]string{
						"id", "technology_id", // Missing columns to cause scan error
					}).AddRow(
						1, technologyID,
					))
			},
			checkResults: func(t *testing.T, results []*TechnologyAlias, err error) {
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
			tt.mockSetup(mockDB, tt.technologyID)

			results, err := repo.ListByTechnologyID(context.Background(), tt.technologyID)
			tt.checkResults(t, results, err)

			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}
