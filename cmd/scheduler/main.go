package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
	"watersystem-ml/internal/config"
	httpinfra "watersystem-ml/internal/infra/http"
	"watersystem-ml/internal/infra/ml"
	"watersystem-ml/internal/infra/tracing"

	"github.com/robfig/cron/v3"
)

const serviceName = "watersystem-ml"

func main() {
	ctx := context.Background()
	log := buildLog()

	conf, err := config.New()
	if err != nil {
		log.ErrorContext(ctx, "error loading config", "error", err)
		os.Exit(1)
	}

	tracingProv, err := tracing.InitTracing(ctx, serviceName)
	if err != nil {
		log.ErrorContext(ctx, "Error initializing tracing", "err", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err = tracingProv.Shutdown(shutdownCtx); err != nil {
			log.ErrorContext(ctx, "Error shutting down tracing", "err", err)
		}
	}()

	//tracer := otel.Tracer(serviceName)

	cron, err := buildCron()
	if err != nil {
		log.ErrorContext(ctx, "Error creating cron", "err", err)
		os.Exit(1)
	}

	go executeTraining(ctx, log)
	go trainingCron(ctx, log, cron)
	go predictionCron(ctx, log, cron)

	serverListener, err := net.Listen("tcp", conf.ServerHost)
	log.InfoContext(ctx, "Starting server", "host", conf.ServerHost)
	if err != nil {
		log.ErrorContext(ctx, "Error starting server", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = serverListener.Close()
	}()

	srv := httpinfra.NewServer(conf.ServerHost)
	defer func() {
		log.InfoContext(ctx, "Closing server")
		_ = srv.Shutdown(ctx)
	}()

	runHTTPServer(ctx, srv, log, serverListener)
}

func trainingCron(ctx context.Context, log *slog.Logger, c *cron.Cron) {
	defer c.Stop()
	_, err := c.AddFunc("* 3 * * *", func() {
		executeTraining(ctx, log)
	})
	if err != nil {
		log.ErrorContext(ctx, "Error adding cron job", "err", err)
		os.Exit(1)
	}
	log.InfoContext(ctx, "Training cron started")
	c.Start()
	<-ctx.Done()
	log.InfoContext(ctx, "Training cron stopped")
}

func executeTraining(ctx context.Context, log *slog.Logger) {
	if err := ml.RunTraining(ctx, log); err != nil {
		log.ErrorContext(ctx, "Error running training", slog.String("error", err.Error()))
	}
}
func predictionCron(ctx context.Context, log *slog.Logger, c *cron.Cron) {
	defer c.Stop()
	_, err := c.AddFunc("00 * * * *", func() {
		predictions, err := ml.RunPrediction(ctx)
		if err != nil {
			log.ErrorContext(ctx, "Error running prediction", slog.String("error", err.Error()))
		}
		for _, pr := range predictions {
			log.InfoContext(ctx, "prediction found",
				slog.String("zone", pr.Zone),
				slog.Bool("should_water", pr.ShouldWater),
				slog.String("decision_reason", pr.DecisionReason),
				slog.Float64("predicted_seconds", pr.PredictedSeconds),
			)
		}
	})
	if err != nil {
		log.ErrorContext(ctx, "Error adding cron job", "err", err)
		os.Exit(1)
	}
	log.InfoContext(ctx, "Prediction cron started")
	c.Start()
	<-ctx.Done()
	log.InfoContext(ctx, "Prediction cron stopped")
}

func buildLog() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	log := slog.New(handler)
	log.With("service", serviceName)
	return log
}

func runHTTPServer(ctx context.Context, srv *http.Server, log *slog.Logger, serverListener net.Listener) {
	go shutdown(ctx, srv, log)

	if err := srv.Serve(serverListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.ErrorContext(ctx, "Error starting server", "err", err)
		os.Exit(1)
	}
}

func shutdown(ctx context.Context, srv *http.Server, log *slog.Logger) {
	<-ctx.Done()
	log.InfoContext(ctx, "Ctrl+C received, shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("error shutting down server", "err", err)
	}
}

func buildCron() (*cron.Cron, error) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return nil, err
	}
	c := cron.New(cron.WithLocation(loc))
	return c, nil
}
