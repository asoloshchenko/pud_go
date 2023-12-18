package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/asoloshchenko/pud_microservice/internal/config"
	"github.com/asoloshchenko/pud_microservice/internal/lib/logger/handlers/slogpretty"
	"github.com/asoloshchenko/pud_microservice/internal/server/handlers/activeINN"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// init config  cleanenv
	cfg := config.ReadConfig()

	// init logger  slog

	log := SetupLogger(cfg.Env)

	log.Debug("debug logging is ON")

	// init router  chi

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	// router.Use(mwLogger.New(log)) разобрать
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/is-not-active", func(r chi.Router) {
		r.Post("/", activeINN.NewCheckINN(log))
	})
	// init server

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("failed to start server")
		}
	}()

	log.Info("server started", slog.Any("addres:", cfg.Address))

	<-done
	log.Info("stopping server")

}

const (
	envLocal = "local"
	envProd  = "prod"
	envDev   = "dev"
)

func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = SetupPrettyLogger()
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

func SetupPrettyLogger() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
