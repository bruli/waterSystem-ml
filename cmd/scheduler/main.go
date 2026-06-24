package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/bruli/go-core/cqs"
	"github.com/bruli/go-core/event"
	"github.com/bruli/watersystem-ml/internal/app"
	"github.com/bruli/watersystem-ml/internal/config"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/bruli/watersystem-ml/internal/domain/watering"
	httpinfra "github.com/bruli/watersystem-ml/internal/infra/http"
	"github.com/bruli/watersystem-ml/internal/infra/influxdb2"
	"github.com/bruli/watersystem-ml/internal/infra/listener"
	"github.com/bruli/watersystem-ml/internal/infra/memory"
	"github.com/bruli/watersystem-ml/internal/infra/python"
	"github.com/bruli/watersystem-ml/internal/infra/tracing"
	watersystem "github.com/bruli/watersystem-ml/internal/infra/water_system"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
)

const serviceName = "watersystem-ml"

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	conf, err := config.New()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return err
	}
	log := buildLog(conf.LogLevel)

	tracingProv, err := tracing.InitTracing(ctx, serviceName)
	if err != nil {
		log.ErrorContext(ctx, "Error initializing tracing", "err", err)
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := tracingProv.Shutdown(shutdownCtx); err != nil {
			log.ErrorContext(ctx, "Error shutting down tracing", "err", err)
		}
	}()

	tracer := otel.Tracer(serviceName)

	duration := 30 * time.Minute
	trainExecutor := python.NewTrainingExecutor(duration, conf.PythonPath, tracer, log)
	predictionRepo := python.NewPredictionRepository(tracer, conf.PythonPath, duration)
	//ntfyPublisher, err := ntfy.NewPublisher(conf.NtfyUser, conf.NtfyPassword, conf.NtfyURL, conf.NtfyTopic, tracer)
	//if err != nil {
	//	log.ErrorContext(ctx, "Error creating ntfy publisher", "err", err)
	//	return err
	//}
	soilMeasureRepo := influxdb2.NewSoilMeasureRepository(conf.InfluxDBURL, conf.InfluxDBToken, conf.InfluxDBOrg, conf.InfluxDBBucket, tracer)
	waterSystemExecutor, err := watersystem.NewExecutor(ctx, 5*time.Second, tracer, conf.WaterSystemHost, conf.WaterSystemPort, conf.WaterSystemToken, log)
	if err != nil {
		log.ErrorContext(ctx, "Error creating water system executor", "err", err)
		return err
	}
	executionRepo := influxdb2.NewExecutionRepository(conf.InfluxDBURL, conf.InfluxDBToken, conf.InfluxDBOrg, conf.InfluxDBBucket, tracer)
	humidityRepo := memory.NewHumidityReferenceRepository(conf.BonsaiBigV100, conf.BonsaiBigV40, conf.BonsaiSmallV100, conf.BonsaiSmallV40)

	trainSvc := ml.NewTrain(trainExecutor, tracer)
	executeSvc := watering.NewExecute(waterSystemExecutor, tracer)
	calculateSvc := ml.NewCalculate(predictionRepo, soilMeasureRepo, humidityRepo, executionRepo, waterSystemExecutor, tracer, func() time.Time {
		loc, err := time.LoadLocation("Europe/Madrid")
		if err != nil {
			log.ErrorContext(ctx, "Error loading location", "err", err)
		}
		return time.Now().In(loc)
	})
	logChMiddleware := cqs.NewCommandHndErrorMiddleware(log, tracer)

	commandBus := cqs.NewCommandBus()
	commandBus.Subscribe(app.ExecuteWateringCommandName, logChMiddleware(app.NewExecuteWatering(executeSvc)))
	execWatOnWaterSysList := listener.NewExecuteWateringOnWatersystemWateringRequested(commandBus)

	eventBus := event.NewBus()
	eventBus.Subscribe(ml.WateringRequestedEventName, execWatOnWaterSysList)

	listenerMdw := cqs.NewCommandHandlerEventBusMiddleware(new(eventBus), tracer)
	multiCHMdw := cqs.CommandHandlerMultiMiddleware(logChMiddleware, listenerMdw)

	calculateWatCh := multiCHMdw(app.NewCalculateWatering(calculateSvc))

	commandBus.Subscribe(app.CalculateWateringEventName, calculateWatCh)

	cronJob, err := buildCron()
	if err != nil {
		log.ErrorContext(ctx, "Error creating cron", "err", err)
		return err
	}

	errCh := make(chan error)
	defer close(errCh)

	go initialTraining(ctx, conf, log, trainSvc)
	go func(ch chan error) {
		if err := trainingCron(ctx, log, cronJob, trainSvc); err != nil {
			log.ErrorContext(ctx, "Error adding cron job", "err", err)
			ch <- err
		}
	}(errCh)
	go runCalculateWatering(ctx, commandBus)

	serverListener, err := net.Listen("tcp", conf.ServerHost)
	log.InfoContext(ctx, "Starting server", "host", conf.ServerHost)
	if err != nil {
		log.ErrorContext(ctx, "Error starting server", "err", err)
		return err
	}
	defer func() {
		_ = serverListener.Close()
	}()

	srv := httpinfra.NewServer(conf.ServerHost)
	defer func() {
		log.InfoContext(ctx, "Closing server")
		_ = srv.Shutdown(ctx)
	}()

	go func(ch chan error) {
		if err := runHTTPServer(ctx, srv, log, serverListener); err != nil {
			ch <- err
		}
	}(errCh)

	if err := <-errCh; err != nil {
		return err
	}
	return nil
}

func initialTraining(ctx context.Context, conf *config.Config, log *slog.Logger, svc *ml.Train) {
	exists, empty, err := checkDir(conf.ModelDir)
	if err != nil {
		log.ErrorContext(ctx, "Error checking model dir", "err", err)
		return
	}

	if exists && !empty {
		log.InfoContext(ctx, "Model dir exists and is not empty, ignoring initial training")
		return
	}

	log.InfoContext(ctx, "Model dir is empty or does not exist, run initial training")
	executeTraining(ctx, log, svc)
}

func trainingCron(ctx context.Context, log *slog.Logger, c *cron.Cron, svc *ml.Train) error {
	defer c.Stop()
	_, err := c.AddFunc("* 3 * * *", func() {
		executeTraining(ctx, log, svc)
	})
	if err != nil {
		return err
	}
	log.InfoContext(ctx, "Training cron started")
	c.Start()
	<-ctx.Done()
	log.InfoContext(ctx, "Training cron stopped")
	return nil
}

func executeTraining(ctx context.Context, log *slog.Logger, svc *ml.Train) {
	if err := svc.Run(ctx); err != nil {
		log.ErrorContext(ctx, "Error running training", slog.String("error", err.Error()))
	}
}

func checkDir(path string) (exists, empty bool, err error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, true, nil
		}
		return false, false, err
	}

	for _, entry := range entries {
		if entry.Name() == "lost+found" {
			continue
		}
		return true, false, nil
	}

	return true, true, nil
}

func runCalculateWatering(ctx context.Context, ch cqs.CommandHandler) {
	tick := time.NewTicker(15 * time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			runCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
			_, _ = ch.Handle(runCtx, app.CalculateWateringCommand{})
			cancel()
		}
	}
}

func buildLog(level string) *slog.Logger {
	levels := map[string]slog.Level{
		"info":  slog.LevelInfo,
		"debug": slog.LevelDebug,
		"error": slog.LevelError,
		"warn":  slog.LevelWarn,
	}
	l := levels[level]

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: l,
	})

	log := slog.New(handler)
	log.With("service", serviceName)
	return log
}

func runHTTPServer(ctx context.Context, srv *http.Server, log *slog.Logger, serverListener net.Listener) error {
	go shutdown(ctx, srv, log)

	if err := srv.Serve(serverListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.ErrorContext(ctx, "Error starting server", "err", err)
		return err
	}
	return nil
}

func shutdown(ctx context.Context, srv *http.Server, log *slog.Logger) {
	<-ctx.Done()
	log.InfoContext(ctx, "Ctrl+C received, shutting down server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
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
