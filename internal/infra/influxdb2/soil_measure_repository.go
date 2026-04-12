package influxdb2

import (
	"context"
	"fmt"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/bruli/watersystem-ml/internal/ptr"
	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"go.opentelemetry.io/otel/trace"
)

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
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement =~ /soil_moisture/)
  |> filter(fn: (r) => r._field == "value")
  |> group(columns: ["entity_id"])
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
		zone, ok := entity.(string)
		if !ok {
			return nil, fmt.Errorf("error parsing soil moisture entity_id: %s", err)
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
