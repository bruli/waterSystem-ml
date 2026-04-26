package influxdb2

import (
	"context"
	"fmt"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/bruli/watersystem-ml/internal/ptr"
	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"go.opentelemetry.io/otel/trace"
)

var zones = map[string]string{
	"sensor.bonsai_big_bonsai_big_soil_moisture_voltage":     "Bonsai big",
	"sensor.bonsai_small_bonsai_small_soil_moisture_voltage": "Bonsai small",
}

type SoilMeasureRepository struct {
	client influxdb.Client
	org    string
	bucket string
	tracer trace.Tracer
}

func (s SoilMeasureRepository) Get(ctx context.Context) ([]ml.SoilMeasure, error) {
	ctx, span := s.tracer.Start(ctx, "SoilMeasureRepository.Get")
	defer span.End()

	query := `
measurements = [
  "sensor.bonsai_big_bonsai_big_soil_moisture_voltage",
  "sensor.bonsai_small_bonsai_small_soil_moisture_voltage"
]

from(bucket: "bonsai-data")
  |> range(start: -40m)
  |> filter(fn: (r) => contains(set: measurements, value: r._measurement))
  |> filter(fn: (r) => r._field == "value")
  |> filter(fn: (r) => r._value >= 0.5 and r._value <= 3.3)
  |> group(columns: ["_measurement"])
  |> median()
`

	result, err := s.client.QueryAPI(s.org).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying soil moisture in influxdb: %w", err)
	}

	measures := make([]ml.SoilMeasure, 0)

	for result.Next() {
		record := result.Record()

		measurement, ok := record.ValueByKey("_measurement").(string)
		if !ok {
			return nil, fmt.Errorf("error parsing soil moisture measurement")
		}

		zone, ok := zones[measurement]
		if !ok {
			return nil, fmt.Errorf("invalid zone: %s", measurement)
		}

		humidity, ok := record.Value().(float64)
		if !ok {
			return nil, fmt.Errorf("error parsing soil moisture value")
		}

		measures = append(measures, ptr.FromPointer(ml.NewSoilMeasure(zone, humidity)))
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("error reading soil moisture result: %w", result.Err())
	}

	return measures, nil
}

func NewSoilMeasureRepository(url, token, org, bucket string, tracer trace.Tracer) *SoilMeasureRepository {
	client := influxdb.NewClient(url, token)
	return &SoilMeasureRepository{client: client, org: org, bucket: bucket, tracer: tracer}
}
