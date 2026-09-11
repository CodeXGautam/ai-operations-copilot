package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"ai-operations-copilot/internal/config"
	"ai-operations-copilot/internal/handler"
	"ai-operations-copilot/internal/llm"
	"ai-operations-copilot/internal/middleware"
	"ai-operations-copilot/internal/query"
	"ai-operations-copilot/internal/repo"
	"ai-operations-copilot/internal/service"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		log.Fatalf("create database directory: %v", err)
	}
	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	migration, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		log.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(string(migration)); err != nil {
		log.Fatalf("run migration: %v", err)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.RequestID())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	store := repo.NewStore(db)
	queryService := &service.QueryService{
		LLM:            llm.NewOpenRouter(cfg.OpenRouterAPIKey, cfg.OpenRouterModel, cfg.OpenRouterBaseURL),
		Context:        &query.ContextBuilder{Orders: store.Orders(), Payments: store.Payments(), Deliveries: store.Deliveries(), Customers: store.Customers()},
		MaxQueryLength: cfg.MaxQueryLength,
		LLMTimeout:     cfg.LLMTimeout,
	}
	queryHandler := &handler.QueryHandler{Service: queryService}
	router.POST("/api/v1/query", queryHandler.Query)

	log.Printf("AI Operations Copilot listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
