package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/app"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	shareddb "github.com/kyle/scout/open-core/db"
)

func main() {
	cfg := app.LoadConfig()
	sqlDB, err := shareddb.Open(context.Background(), shareddb.Config{
		Dialect:    shareddb.DialectSQLite,
		SQLitePath: cfg.DBPath,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	application := app.New(cfg, sqlDB)

	// Run startup tasks: reconcile orphaned runs, council reconciliation, optional demo seed.
	aiSvc := ai.NewService(repository.New(sqlDB))
	if err := services.RunStartupTasks(context.Background(), application.Repo, aiSvc); err != nil {
		log.Printf("startup: reconciliation warning: %v", err)
	}

	// Start the cron scheduler: load persisted enabled schedules then begin ticking.
	if err := application.Scheduler.LoadAll(context.Background()); err != nil {
		log.Printf("scheduler: failed to load schedules: %v", err)
	}
	application.Scheduler.Start()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           application.Handler(),
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       10 * time.Minute,
		WriteTimeout:      10 * time.Minute,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Scout Go API running at http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	if err := application.Scheduler.Stop(ctx); err != nil {
		log.Printf("Scheduler stop: %v", err)
	}
	log.Println("Server exited cleanly")
}
