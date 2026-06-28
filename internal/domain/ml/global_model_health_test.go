package ml_test

import (
	"testing"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/stretchr/testify/require"
)

func TestGlobalModelHealth_Check(t *testing.T) {
	type fields struct {
		zones []*ml.ModelHealth
	}
	tests := []struct {
		name               string
		fields             fields
		expectedEventZones []string
	}{
		{
			name: "when all zones are healthy, no events are returned",
			fields: fields{
				zones: []*ml.ModelHealth{
					ml.NewModelHealth(bonsaiBigZone, 18, 2),
					ml.NewModelHealth(bonsaiSmallZone, 18, 2),
				},
			},
			expectedEventZones: []string{},
		},
		{
			name: "when all zones are unhealthy, two events are returned",
			fields: fields{
				zones: []*ml.ModelHealth{
					ml.NewModelHealth(bonsaiBigZone, 11, 9),
					ml.NewModelHealth(bonsaiSmallZone, 11, 9),
				},
			},
			expectedEventZones: []string{bonsaiBigZone, bonsaiSmallZone},
		},
		{
			name: "when bonsai big zone is unhealthy, one event is returned",
			fields: fields{
				zones: []*ml.ModelHealth{
					ml.NewModelHealth(bonsaiBigZone, 11, 9),
					ml.NewModelHealth(bonsaiSmallZone, 19, 1),
				},
			},
			expectedEventZones: []string{bonsaiBigZone},
		},
		{
			name: "when bonsai small zone is unhealthy, one event is returned",
			fields: fields{
				zones: []*ml.ModelHealth{
					ml.NewModelHealth(bonsaiBigZone, 19, 1),
					ml.NewModelHealth(bonsaiSmallZone, 11, 9),
				},
			},
			expectedEventZones: []string{bonsaiSmallZone},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a GlobalModelHealth struct,
		when Check method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			gmh := ml.NewGlobalModelHealth(tt.fields.zones)
			gmh.Check()
			events := gmh.Events()
			zonesWithEvents := make([]string, len(tt.expectedEventZones))
			for i, ev := range events {
				zev, ok := ev.(*ml.ZoneModelDegradedEvent)
				require.True(t, ok)
				zonesWithEvents[i] = zev.Zone
			}
			require.Equal(t, tt.expectedEventZones, zonesWithEvents)
		})
	}
}
