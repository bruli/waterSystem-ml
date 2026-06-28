package ml

import "github.com/bruli/go-core/event"

type GlobalModelHealth struct {
	event.BasicAggregateRoot
	zones map[string]*ModelHealth
}

func (g *GlobalModelHealth) Check() {
	for name, zone := range g.zones {
		if zone.isDegraded() {
			g.Record(NewZoneModelDegradedEvent(name))
		}
	}
}

func NewGlobalModelHealth(zones []*ModelHealth) *GlobalModelHealth {
	zonesMap := make(map[string]*ModelHealth)
	for _, zone := range zones {
		zonesMap[zone.Zone()] = zone
	}
	return &GlobalModelHealth{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		zones:              zonesMap,
	}
}
