// Package router provides HTTP routing configuration and middleware setup for the job board API.
// It handles CORS, Swagger documentation, and API route registration using the Gin web framework.
package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/rodruizronald/ticos-in-tech/internal/config"
	"github.com/rodruizronald/ticos-in-tech/internal/jobs"
)

// Router handles HTTP routing and middleware setup
type Router struct {
	jobRepos jobs.DataRepository
}

// New creates a new Router instance with dependencies
func New(jobRepos jobs.DataRepository) *Router {
	return &Router{
		jobRepos: jobRepos,
	}
}

// Setup configures and returns a Gin engine with all middleware and routes
func (r *Router) Setup(cfg *config.GinConfig) *gin.Engine {
	engine := gin.Default()
	gin.SetMode(cfg.Mode)

	r.setupMiddleware(engine)
	r.setupRoutes(engine)

	return engine
}

// setupMiddleware configures all middleware
func (r *Router) setupMiddleware(engine *gin.Engine) {
	// Add CORS middleware
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // React app URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
}

// setupRoutes configures all application routes
func (r *Router) setupRoutes(engine *gin.Engine) {
	// Swagger routes
	r.setupSwaggerRoutes(engine)

	// API routes
	r.setupAPIRoutes(engine)
}

// setupSwaggerRoutes configures Swagger documentation routes
func (r *Router) setupSwaggerRoutes(engine *gin.Engine) {
	if gin.Mode() != gin.ReleaseMode {
		engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}

// setupAPIRoutes configures all API routes
func (r *Router) setupAPIRoutes(engine *gin.Engine) {
	v1 := engine.Group("/api/v1")

	// Job routes
	jobHandler := jobs.NewHandler(r.jobRepos)
	jobHandler.RegisterRoutes(v1)
}
