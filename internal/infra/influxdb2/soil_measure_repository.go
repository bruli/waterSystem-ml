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
	"bonsai_big_bonsai_big_soil_moisture_voltage": "Bonsai big",
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
from(bucket: "bonsai-data")
  |> range(start: -24h)
  |> filter(fn: (r) => r._measurement == "sensor.bonsai_big_bonsai_big_soil_moisture_voltage")
  |> filter(fn: (r) => r._field == "value")
  |> last()
`
	result, err := s.client.QueryAPI(s.org).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying soil moisture in influxdb: %s", err)
	}
	measures := make([]ml.SoilMeasure, 0)
	for result.Next() {
		record := result.Record()

		entity := record.ValueByKey("entity_id")
		zoneFormated, ok := entity.(string)
		if !ok {
			return nil, fmt.Errorf("error parsing soil moisture entity_id: %s", err)
		}
		zone, ok := zones[zoneFormated]
		if !ok {
			return nil, fmt.Errorf("invalid zone: %s", zoneFormated)
		}
		value := record.Value()
		humidity, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("error parsing soil moisture value: %s", err)
		}
		measures = append(measures, ptr.FromPointer(ml.NewSoilMeasure(zone, humidity)))
	}
	return measures, nil
}

func NewSoilMeasureRepository(url, token, org, bucket string, tracer trace.Tracer) *SoilMeasureRepository {
	client := influxdb.NewClient(url, token)
	return &SoilMeasureRepository{client: client, org: org, bucket: bucket, tracer: tracer}
}
