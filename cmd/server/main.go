// @title Job Board API
// @version 1.0
// @description A job board API for managing job postings
// @contact.name API Support
// @contact.email support@example.com
// @host localhost:8080
// @BasePath /api/v1
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/sync/errgroup"

	_ "github.com/rodruizronald/ticos-in-tech/docs"
	"github.com/rodruizronald/ticos-in-tech/internal/database"
	"github.com/rodruizronald/ticos-in-tech/internal/jobs"
	"github.com/rodruizronald/ticos-in-tech/internal/jobtech"
)

func main() {
	var err error
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		stop()
		if err != nil {
			os.Exit(1)
		}
	}()
	err = run(ctx)
}

func run(ctx context.Context) error {
	// Initialize logger
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Get database config
	dbConfig := database.DefaultConfig()

	// Connect to the database
	dbpool, err := database.Connect(ctx, &dbConfig)
	if err != nil {
		log.Errorf("Unable to connect to database: %v", err)
		return err
	}
	defer dbpool.Close()

	// Initialize Gin
	r := gin.Default()

	gin.SetMode(gin.DebugMode)

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // React app URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Swagger endpoint
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// API routes
	v1 := r.Group("/api/v1")

	jobRepo := jobs.NewRepository(dbpool)
	jobtechRepo := jobtech.NewRepository(dbpool)
	jobRepos := jobs.NewRepositories(jobRepo, jobtechRepo)
	jobHandler := jobs.NewHandler(jobRepos)
	jobHandler.RegisterRoutes(v1)

	port := "8080"
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Create error group with context
	g, gCtx := errgroup.WithContext(ctx)

	// Start HTTP server in goroutine
	g.Go(func() error {
		log.Printf("Server starting on port %s", port)
		log.Printf("Swagger UI available at: http://localhost:%s/swagger/index.html", port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Server failed to start: %v", err)
			return err
		}
		return nil
	})

	// Handle graceful shutdown in another goroutine
	g.Go(func() error {
		<-gCtx.Done() // Wait for context cancellation (SIGINT/SIGTERM)

		log.Println("Shutting down server...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Errorf("Server forced to shutdown: %v", err)
			return err
		}

		log.Println("Server exited gracefully")
		return nil
	})

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		log.Errorf("Application error: %v", err)
		return err
	}

	return nil
}
