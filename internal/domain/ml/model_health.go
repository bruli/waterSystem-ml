package ml

import "github.com/bruli/go-core/event"

const DefaultMaxFailureRate = 0.35

type ModelHealth struct {
	event.BasicAggregateRoot
	zone                  string
	successfulPredictions int
	failedPredictions     int
	maxFailureRate        float64
}

func (m *ModelHealth) Zone() string {
	return m.zone
}

func (m *ModelHealth) totalPredictions() int {
	return m.successfulPredictions + m.failedPredictions
}

func (m *ModelHealth) failureRate() float64 {
	total := m.totalPredictions()
	if total == 0 {
		return 0
	}

	return float64(m.failedPredictions) / float64(total)
}

func (m *ModelHealth) isDegraded() bool {
	if m.totalPredictions() == 0 {
		return false
	}

	return m.failureRate() >= m.maxFailureRate
}

func (m *ModelHealth) Check() {
	if m.isDegraded() {
		m.Record(NewZoneModelDegradedEvent(m.zone))
	}
}

func NewModelHealth(zone string, successfulPredictions, failedPredictions int) *ModelHealth {
	return &ModelHealth{
		BasicAggregateRoot:    event.BasicAggregateRoot{},
		zone:                  zone,
		successfulPredictions: successfulPredictions,
		failedPredictions:     failedPredictions,
		maxFailureRate:        DefaultMaxFailureRate,
	}
}
