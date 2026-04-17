package watersystem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bruli/watersystem-ml/internal/domain/watering"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var zones = map[string]string{
	"Bonsai big":   "bb",
	"Bonsai small": "bs",
}

type ExecuteBody struct {
	Seconds int `json:"seconds"`
}

type Executor struct {
	cl                *http.Client
	tracer            trace.Tracer
	host, port, token string
}

func (e Executor) Execute(ctx context.Context, w *watering.Watering) error {
	ctx, span := e.tracer.Start(ctx, "WaterSystem.Execute")
	defer span.End()
	zone, ok := zones[w.Zone()]
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", e.token)

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

func NewExecutor(timeout time.Duration, tracer trace.Tracer, host, port, token string) *Executor {
	cl := http.Client{Timeout: timeout}
	return &Executor{tracer: tracer, host: host, port: port, cl: &cl, token: host}
}
