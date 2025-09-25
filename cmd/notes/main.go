package main

import (
	"log/slog"
	"net/http"
	"notes/internal/config"
	"notes/internal/handlers/note/delete"
	"notes/internal/handlers/note/get"
	"notes/internal/handlers/note/getall"
	noteSave "notes/internal/handlers/note/save"
	"notes/internal/handlers/note/update"
	"notes/internal/handlers/user/login"
	userSave "notes/internal/handlers/user/save"
	"notes/internal/storage/postgres"
	"notes/pkg/logger/handlers/slogpretty"
	"notes/pkg/logger/sl"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	JWTMiddleware "notes/internal/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.Load()
	log := setupLogger(cfg.Env)

	log.Info("starting notes service", slog.String("env", cfg.Env))
	log.Debug("debug log enabled")
	storage, err := postgres.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}
	_ = storage
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Post("/users/register", userSave.New(log, storage))
	router.Post("/users/login", login.New(log, storage))

	router.Route("/users/{id}/notes", func(r chi.Router) {
		r.Use(JWTMiddleware.JWT)
		r.Post("/", noteSave.New(log, storage))
		r.Get("/", getall.New(log, storage))
		r.Get("/{note_id}", get.New(log, storage))
		r.Put("/{note_id}", update.New(log, storage))
		r.Delete("/{note_id}", delete.New(log, storage))
	})

	log.Info("starting server", slog.String("address", cfg.Address))
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}
	handler := opts.NewPrettyHandler(os.Stdout)
	return slog.New(handler)
}
