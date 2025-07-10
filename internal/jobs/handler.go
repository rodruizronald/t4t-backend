package jobs

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/rodruizronald/ticos-in-tech/internal/httpservice"
	"github.com/rodruizronald/ticos-in-tech/internal/jobtech"
)

// Constants for job routes and endpoints
const (
	JobsRoute = "/jobs"
)

// DataRepository interface to make database operations for the Job model.
type DataRepository interface {
	SearchJobsWithCount(ctx context.Context, params *SearchParams) ([]*JobWithCompany, int, error)
	GetJobTechnologiesBatch(ctx context.Context, jobIDs []int) (map[int][]*jobtech.JobTechnologyWithDetails, error)
}

// Repositories struct to hold repositories for job and jobtech models
type Repositories struct {
	jobRepo     *Repository
	jobtechRepo *jobtech.Repository
}

// SearchJobsWithCount delegates to the job repository's SearchJobsWithCount method
func (r *Repositories) SearchJobsWithCount(ctx context.Context, params *SearchParams) ([]*JobWithCompany, int, error) {
	return r.jobRepo.SearchJobsWithCount(ctx, params)
}

// GetJobTechnologiesBatch delegates to the jobtech repository's GetJobTechnologiesBatch method
func (r *Repositories) GetJobTechnologiesBatch(ctx context.Context, jobIDs []int) (
	map[int][]*jobtech.JobTechnologyWithDetails, error) {
	return r.jobtechRepo.GetJobTechnologiesBatch(ctx, jobIDs)
}

// Handler handles HTTP requests for job operations using the generic httpservice
type Handler struct {
	searchHandler *httpservice.SearchHandler[*SearchRequest, *SearchParams, JobResponseList]
}

// NewRepositories creates a new job and jobtech repositories
func NewRepositories(jobRepo *Repository, jobtechRepo *jobtech.Repository) *Repositories {
	return &Repositories{jobRepo: jobRepo, jobtechRepo: jobtechRepo}
}

// NewHandler creates a new job handler using httpservice.NewSearchHandlerWithDefaults
func NewHandler(repos DataRepository) *Handler {
	// Create the search service
	searchService := NewSearchService(repos)

	// Create the generic search handler with defaults
	searchHandler := httpservice.NewSearchHandlerWithDefaults(
		func() *SearchRequest { return &SearchRequest{} }, // Request factory function
		searchService,
	)

	return &Handler{
		searchHandler: searchHandler,
	}
}

// RegisterRoutes registers job routes with the given router group
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET(JobsRoute, h.SearchJobs)
}

// SearchJobs godoc
// @Summary Search for jobs
// @Description Search for jobs with optional filters and pagination
// @Tags jobs
// @Accept json
// @Produce json
// @Param q query string true "Search query" example("golang developer")
// @Param limit query int false "Number of results to return (max 100)" default(20) example(20)
// @Param offset query int false "Number of results to skip" default(0) example(0)
// @Param experience_level query string false "Experience level filter" \
// Enums(Entry-level,Junior,Mid-level,Senior,Lead,Principal,Executive) example("Senior")
// @Param employment_type query string false "Employment type filter" \
// Enums(Full-time,Part-time,Contract,Freelance,Temporary,Internship) example("Full-time")
// @Param location query string false "Location filter" Enums(Costa Rica,LATAM) example("Costa Rica")
// @Param work_mode query string false "Work mode filter" Enums(Remote,Hybrid,Onsite) example("Remote")
// @Param company query string false "Company name filter (partial match)" example("Tech Corp")
// @Param date_from query string false "Start date filter (YYYY-MM-DD)" example("2024-01-01")
// @Param date_to query string false "End date filter (YYYY-MM-DD)" example("2024-12-31")
// @Success 200 {object} SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /jobs [get]
func (h *Handler) SearchJobs(c *gin.Context) { h.searchHandler.HandleSearch(c) }
