package watersystem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/bruli/watersystem-ml/internal/domain/watering"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ExecuteBody struct {
	Seconds int `json:"seconds"`
}

type Executor struct {
	cl                *http.Client
	tracer            trace.Tracer
	host, port, token string
	zones             map[string]string
	logger            *slog.Logger
}

func (e *Executor) Execute(ctx context.Context, w *watering.Watering) error {
	ctx, span := e.tracer.Start(ctx, "WaterSystem.Execute")
	defer span.End()
	zone, ok := e.zones[w.Zone()]
	if !ok {
		err := fmt.Errorf("invalid zone: %s", w.Zone())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	url := fmt.Sprintf("%s:%s/zones%s/execute", e.host, e.port, zone)
	body := ExecuteBody{Seconds: w.Seconds()}
	data, err := json.Marshal(body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error marshaling body: %s", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error creating request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", e.token)

	resp, err := e.cl.Do(req)
	defer func() {
		_ = resp.Body.Close()
	}()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error executing request: %s", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		err := fmt.Errorf("error executing request: %s", resp.Status)
		span.RecordError(err)
		span.SetStatus(codes.Error, "error executing request")
		return err
	}
	span.SetStatus(codes.Ok, "OK")
	return nil
}

func (e *Executor) getZones(ctx context.Context) error {
	ctx, span := e.tracer.Start(ctx, "WaterSystem.getZones")
	defer span.End()
	url := fmt.Sprintf("%s:%s/zones", e.host, e.port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error creating request: %s", err)
	}
	req.Header.Add("Authorization", e.token)

	resp, err := e.cl.Do(req)
	if err != nil {
		err := fmt.Errorf("error executing request: %s", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("error executing request: %s", resp.Status)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	var zonesBody []Zone
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error reading response: %s", err)
	}
	if err := json.Unmarshal(body, &zonesBody); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("error decoding response: %s", err)
	}
	e.logger.InfoContext(ctx, "Zones found", slog.Int("count", len(zonesBody)))
	zones := make(map[string]string)
	for _, z := range zonesBody {
		e.logger.InfoContext(ctx, "Zone", slog.String("name", z.Name), slog.String("id", z.ID))
		zones[z.Name] = z.ID
	}
	e.zones = zones
	span.SetStatus(codes.Ok, "OK")
	return nil
}

func NewExecutor(ctx context.Context, timeout time.Duration, tracer trace.Tracer, host, port, token string, log *slog.Logger) (*Executor, error) {
	cl := http.Client{Timeout: timeout}
	ex := Executor{tracer: tracer, host: host, port: port, cl: &cl, token: token, logger: log}
	if err := ex.getZones(ctx); err != nil {
		return nil, err
	}
	return &ex, nil
}
